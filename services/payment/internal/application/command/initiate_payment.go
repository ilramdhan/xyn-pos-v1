package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// InitiatePaymentInput is the request DTO for starting a payment.
type InitiatePaymentInput struct {
	IdempotencyKey string
	TenantID       uuid.UUID
	OrderID        uuid.UUID
	Amount         int64 // sen
	Method         payment.PaymentMethod
	CustomerName   string
	CustomerEmail  string
}

// InitiatePaymentHandler orchestrates payment initiation.
type InitiatePaymentHandler struct {
	repo      payment.PaymentRepository
	gateway   payment.PaymentGateway
	idemStore IdempotencyStore
}

// NewInitiatePaymentHandler wires the handler.
func NewInitiatePaymentHandler(
	repo payment.PaymentRepository,
	gateway payment.PaymentGateway,
	idemStore IdempotencyStore,
) *InitiatePaymentHandler {
	return &InitiatePaymentHandler{repo: repo, gateway: gateway, idemStore: idemStore}
}

// Handle initiates a payment. Idempotent: returns existing payment if key already used.
func (h *InitiatePaymentHandler) Handle(ctx context.Context, in InitiatePaymentInput) (*payment.Payment, error) {
	// 1. Check PostgreSQL-level idempotency first (persistent).
	existing, err := h.repo.FindByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey)
	if err != nil && !errors.Is(err, payment.ErrPaymentNotFound) {
		return nil, fmt.Errorf("InitiatePayment.FindByIdempotencyKey: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// 2. Redis SETNX guard — prevents duplicate gateway calls under concurrent requests.
	redisKey := fmt.Sprintf("payment:idem:%s:%s", in.TenantID.String(), in.IdempotencyKey)
	acquired, err := h.idemStore.Acquire(ctx, redisKey)
	if err != nil {
		// Non-fatal: Redis failure degrades to DB-only idempotency.
		slog.WarnContext(ctx, "idempotency store unavailable", "err", err)
	}

	// 3. Create domain aggregate.
	p, err := payment.NewPayment(in.TenantID, in.OrderID, in.Amount, in.Method, in.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("InitiatePayment.NewPayment: %w", err)
	}

	// 4. Call gateway.
	result, err := h.gateway.CreateTransaction(ctx, p, in.CustomerName, in.CustomerEmail)
	if err != nil {
		// Release Redis key on gateway failure so caller can retry.
		if acquired {
			_ = h.idemStore.Release(ctx, redisKey)
		}
		return nil, fmt.Errorf("InitiatePayment.CreateTransaction: %w", payment.ErrGatewayFailure)
	}
	p.SetGatewayResult(result)

	// 5. Persist.
	if err := h.repo.Save(ctx, p); err != nil {
		if errors.Is(err, payment.ErrDuplicateIdempotencyKey) {
			// Race: another goroutine already saved. Return existing.
			return h.repo.FindByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey)
		}
		return nil, fmt.Errorf("InitiatePayment.Save: %w", err)
	}
	return p, nil
}
