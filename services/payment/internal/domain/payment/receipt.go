package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Receipt is the fiscal document issued after a successful payment.
type Receipt struct {
	ID            uuid.UUID
	PaymentID     uuid.UUID
	OrderID       uuid.UUID
	TenantID      uuid.UUID
	ReceiptNumber string
	Amount        int64
	Method        PaymentMethod
	IssuedAt      time.Time
	events        []DomainEvent
}

// ReceiptRepository is the persistence port for Receipt aggregates.
type ReceiptRepository interface {
	Save(ctx context.Context, r *Receipt) error
	FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*Receipt, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) (*Receipt, error)
}

// GenerateReceipt creates a Receipt from a successful Payment.
func GenerateReceipt(p *Payment) (*Receipt, error) {
	if p.Status != StatusSuccess {
		return nil, fmt.Errorf("GenerateReceipt: %w", ErrReceiptNotAllowed)
	}
	now := time.Now().UTC()
	r := &Receipt{
		ID:            uuid.New(),
		PaymentID:     p.ID,
		OrderID:       p.OrderID,
		TenantID:      p.TenantID,
		ReceiptNumber: fmt.Sprintf("RCP-%s", p.ID.String()[:8]),
		Amount:        p.Amount,
		Method:        p.Method,
		IssuedAt:      now,
	}
	r.events = append(r.events, ReceiptIssuedEvent{
		ReceiptID:     r.ID,
		PaymentID:     r.PaymentID,
		OrderID:       r.OrderID,
		TenantID:      r.TenantID,
		ReceiptNumber: r.ReceiptNumber,
		Amount:        r.Amount,
		Method:        r.Method,
		OccurredAt:    now,
	})
	return r, nil
}

// PopEvents returns and clears pending domain events.
func (r *Receipt) PopEvents() []DomainEvent {
	evs := r.events
	r.events = nil
	return evs
}
