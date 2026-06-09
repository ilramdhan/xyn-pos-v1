package payment

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus is the lifecycle state of a payment.
type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusSuccess  PaymentStatus = "success"
	StatusFailed   PaymentStatus = "failed"
	StatusVoided   PaymentStatus = "voided"
	StatusRefunded PaymentStatus = "refunded"
)

// PaymentMethod is the payment channel used.
type PaymentMethod string

const (
	MethodQRIS         PaymentMethod = "qris"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodCreditCard   PaymentMethod = "credit_card"
	MethodCash         PaymentMethod = "cash"
)

// Payment is the aggregate root for the payment bounded context.
type Payment struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	OrderID         uuid.UUID
	Amount          int64 // minor units (sen)
	Status          PaymentStatus
	Method          PaymentMethod
	ExternalID      string // Midtrans transaction ID
	SnapToken       string
	SnapRedirectURL string
	IdempotencyKey  string
	RefundedAmount  int64  // minor units; 0 until a refund is processed
	RefundReason    string // human-readable reason
	CreatedAt       time.Time
	UpdatedAt       time.Time
	events          []DomainEvent
}

// NewPayment creates a new payment in PENDING status.
func NewPayment(tenantID, orderID uuid.UUID, amount int64, method PaymentMethod, idempotencyKey string) (*Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	now := time.Now().UTC()
	return &Payment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		OrderID:        orderID,
		Amount:         amount,
		Status:         StatusPending,
		Method:         method,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// SetGatewayResult stores the Snap token and external ID returned by the gateway.
func (p *Payment) SetGatewayResult(result *GatewayResult) {
	p.ExternalID = result.ExternalID
	p.SnapToken = result.SnapToken
	p.SnapRedirectURL = result.SnapRedirectURL
	p.UpdatedAt = time.Now().UTC()
}

// MarkSuccess transitions from PENDING to SUCCESS and emits PaymentCompletedEvent.
func (p *Payment) MarkSuccess(externalID string) error {
	if p.Status != StatusPending {
		return fmt.Errorf("MarkSuccess: %w", ErrInvalidStatusTransition)
	}
	p.Status = StatusSuccess
	p.ExternalID = externalID
	p.UpdatedAt = time.Now().UTC()
	p.events = append(p.events, PaymentCompletedEvent{
		PaymentID:  p.ID,
		OrderID:    p.OrderID,
		TenantID:   p.TenantID,
		Amount:     p.Amount,
		Method:     p.Method,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

// MarkFailed transitions from PENDING to FAILED.
func (p *Payment) MarkFailed() error {
	if p.Status != StatusPending {
		return fmt.Errorf("MarkFailed: %w", ErrInvalidStatusTransition)
	}
	p.Status = StatusFailed
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Void transitions from SUCCESS to VOIDED and emits PaymentVoidedEvent.
func (p *Payment) Void(reason string, voidedBy uuid.UUID) error {
	if p.Status != StatusSuccess {
		return fmt.Errorf("Void: %w", ErrInvalidStatusTransition)
	}
	p.Status = StatusVoided
	p.UpdatedAt = time.Now().UTC()
	p.events = append(p.events, PaymentVoidedEvent{
		PaymentID:  p.ID,
		OrderID:    p.OrderID,
		TenantID:   p.TenantID,
		VoidedBy:   voidedBy,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

// Refund transitions a SUCCESS payment to REFUNDED.
// refundAmount may be less than p.Amount (partial refund), but this is a
// single-shot operation — the status becomes REFUNDED regardless of amount,
// preventing subsequent refunds on the same payment.
func (p *Payment) Refund(refundAmount int64, reason string) error {
	if p.Status != StatusSuccess {
		return fmt.Errorf("Refund: %w", ErrInvalidStatusTransition)
	}
	if refundAmount <= 0 || refundAmount > p.Amount {
		return ErrPartialRefundInvalid
	}
	p.Status = StatusRefunded
	p.RefundedAmount = refundAmount
	p.RefundReason = reason
	p.UpdatedAt = time.Now().UTC()
	p.events = append(p.events, PaymentRefundedEvent{
		PaymentID:      p.ID,
		OrderID:        p.OrderID,
		TenantID:       p.TenantID,
		RefundedAmount: refundAmount,
		OccurredAt:     time.Now().UTC(),
	})
	return nil
}

// PopEvents returns and clears all pending domain events.
func (p *Payment) PopEvents() []DomainEvent {
	evs := p.events
	p.events = nil
	return evs
}
