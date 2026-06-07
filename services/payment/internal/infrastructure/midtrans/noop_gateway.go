package midtrans

import (
	"context"
	"fmt"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// NoopGateway is a test double that always succeeds without calling Midtrans.
type NoopGateway struct{}

var _ payment.PaymentGateway = (*NoopGateway)(nil)

// CreateTransaction returns a fake snap token.
func (g *NoopGateway) CreateTransaction(
	_ context.Context,
	p *payment.Payment,
	_, _ string,
) (*payment.GatewayResult, error) {
	return &payment.GatewayResult{
		ExternalID:      fmt.Sprintf("noop-%s", p.ID.String()),
		SnapToken:       fmt.Sprintf("snap-token-%s", p.ID.String()),
		SnapRedirectURL: fmt.Sprintf("https://app.sandbox.midtrans.com/snap/v2/vtweb/%s", p.ID.String()),
	}, nil
}

// VoidTransaction always succeeds.
func (g *NoopGateway) VoidTransaction(_ context.Context, _ string) error {
	return nil
}
