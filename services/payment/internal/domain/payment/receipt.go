package payment

import (
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

// GenerateReceipt creates a Receipt from a successful Payment.
func GenerateReceipt(p *Payment) (*Receipt, error) {
	if p.Status != StatusSuccess {
		return nil, ErrReceiptNotAllowed
	}
	now := time.Now().UTC()
	receiptID := uuid.New()
	r := &Receipt{
		ID:            receiptID,
		PaymentID:     p.ID,
		OrderID:       p.OrderID,
		TenantID:      p.TenantID,
		ReceiptNumber: fmt.Sprintf("RCP-%s", receiptID.String()[:13]),
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
