package midtrans

import (
	"context"
	"fmt"

	midtranssdk "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// MidtransGateway implements payment.PaymentGateway using Midtrans Snap.
type MidtransGateway struct {
	client snap.Client
}

// NewMidtransGateway creates a gateway with the given server key.
func NewMidtransGateway(serverKey string, isProduction bool) *MidtransGateway {
	env := midtranssdk.Sandbox
	if isProduction {
		env = midtranssdk.Production
	}
	var c snap.Client
	c.New(serverKey, env)
	return &MidtransGateway{client: c}
}

var _ payment.PaymentGateway = (*MidtransGateway)(nil)

// CreateTransaction initiates a Snap payment session.
// Amount is in sen (int64); Midtrans expects integer rupiah (amount / 100).
func (g *MidtransGateway) CreateTransaction(
	ctx context.Context,
	p *payment.Payment,
	customerName, customerEmail string,
) (*payment.GatewayResult, error) {
	req := &snap.Request{
		TransactionDetails: midtranssdk.TransactionDetails{
			OrderID:  p.ID.String(),
			GrossAmt: p.Amount / 100, // sen → rupiah
		},
		CustomerDetail: &midtranssdk.CustomerDetails{
			FName: customerName,
			Email: customerEmail,
		},
		EnabledPayments: methodToEnabledPayments(p.Method),
	}

	resp, merr := g.client.CreateTransaction(req)
	if merr != nil {
		return nil, fmt.Errorf("midtrans.CreateTransaction: %w", merr)
	}

	return &payment.GatewayResult{
		ExternalID:      p.ID.String(),
		SnapToken:       resp.Token,
		SnapRedirectURL: resp.RedirectURL,
	}, nil
}

// VoidTransaction cancels a Midtrans transaction by external ID.
func (g *MidtransGateway) VoidTransaction(_ context.Context, externalID string) error {
	// Midtrans cancel uses core API; not yet wired.
	return fmt.Errorf("midtrans.VoidTransaction: not yet wired (externalID=%s)", externalID)
}

// RefundTransaction requests a refund from Midtrans for a given transaction.
// refundAmount is in minor units (sen).
func (g *MidtransGateway) RefundTransaction(_ context.Context, externalID string, refundAmount int64) error {
	// TODO(phase5): wire Midtrans CoreAPI refund endpoint
	return fmt.Errorf("midtrans.RefundTransaction: not yet wired (externalID=%s, amount=%d)", externalID, refundAmount)
}

// methodToEnabledPayments maps our PaymentMethod to Snap-specific payment types.
// Note: QRIS is not a named constant in midtrans-go v1.3.8; GoPay supports QRIS in sandbox.
func methodToEnabledPayments(m payment.PaymentMethod) []snap.SnapPaymentType {
	switch m {
	case payment.MethodQRIS:
		return []snap.SnapPaymentType{snap.PaymentTypeGopay}
	case payment.MethodBankTransfer:
		return []snap.SnapPaymentType{
			snap.PaymentTypeBCAVA,
			snap.PaymentTypeBNIVA,
			snap.PaymentTypeBRIVA,
		}
	case payment.MethodCreditCard:
		return []snap.SnapPaymentType{snap.PaymentTypeCreditCard}
	default:
		// For cash / unknown: allow all channels (Midtrans decides).
		return nil
	}
}
