package payment

import "context"

// GatewayResult is the outcome of a payment gateway call.
type GatewayResult struct {
	ExternalID      string
	SnapToken       string
	SnapRedirectURL string
}

// PaymentGateway is the port for external payment processors.
type PaymentGateway interface {
	// CreateTransaction initiates a new payment with the gateway.
	CreateTransaction(ctx context.Context, p *Payment, customerName, customerEmail string) (*GatewayResult, error)
	// VoidTransaction cancels an existing gateway transaction.
	VoidTransaction(ctx context.Context, externalID string) error
}
