package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/xyn-pos/gen/go/payment/v1"
	"github.com/xyn-pos/services/payment/internal/application/command"
	"github.com/xyn-pos/services/payment/internal/application/query"
	"github.com/xyn-pos/services/payment/internal/domain/payment"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// PaymentHandler implements paymentv1.PaymentServiceServer.
type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	initiateH   *command.InitiatePaymentHandler
	voidH       *command.VoidPaymentHandler
	getH        *query.GetPaymentHandler
	getByOrderH *query.GetPaymentByOrderHandler
}

// NewPaymentHandler assembles the handler.
func NewPaymentHandler(
	initiateH *command.InitiatePaymentHandler,
	voidH *command.VoidPaymentHandler,
	getH *query.GetPaymentHandler,
	getByOrderH *query.GetPaymentByOrderHandler,
) *PaymentHandler {
	return &PaymentHandler{
		initiateH:   initiateH,
		voidH:       voidH,
		getH:        getH,
		getByOrderH: getByOrderH,
	}
}

// InitiatePayment starts a new payment and returns the snap token.
func (h *PaymentHandler) InitiatePayment(ctx context.Context, req *paymentv1.InitiatePaymentRequest) (*paymentv1.InitiatePaymentResponse, error) {
	tenantID, err := tenantIDFromContext(ctx, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}

	p, err := h.initiateH.Handle(ctx, command.InitiatePaymentInput{
		IdempotencyKey: req.IdempotencyKey,
		TenantID:       tenantID,
		OrderID:        orderID,
		Amount:         req.Amount,
		Method:         protoMethodToDomain(req.Method),
		CustomerName:   req.CustomerName,
		CustomerEmail:  req.CustomerEmail,
	})
	if err != nil {
		return nil, mapPaymentError(err)
	}
	return &paymentv1.InitiatePaymentResponse{Payment: domainToProto(p)}, nil
}

// VoidPayment cancels a successful payment.
func (h *PaymentHandler) VoidPayment(ctx context.Context, req *paymentv1.VoidPaymentRequest) (*paymentv1.VoidPaymentResponse, error) {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}

	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}

	// Load payment first to verify tenant ownership before voiding.
	p, err := h.getH.Handle(ctx, paymentID)
	if err != nil {
		return nil, mapPaymentError(err)
	}
	if p.TenantID != claims.TenantID {
		return nil, status.Error(codes.NotFound, "payment not found")
	}

	if err := h.voidH.Handle(ctx, command.VoidPaymentInput{
		PaymentID: paymentID,
		Reason:    req.Reason,
		VoidedBy:  claims.UserID,
	}); err != nil {
		return nil, mapPaymentError(err)
	}
	return &paymentv1.VoidPaymentResponse{}, nil
}

// GetPayment retrieves a payment by ID.
func (h *PaymentHandler) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	id, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid payment_id")
	}
	p, err := h.getH.Handle(ctx, id)
	if err != nil {
		return nil, mapPaymentError(err)
	}
	// Verify tenant ownership — return NotFound to avoid existence oracle.
	if claims, ok := claimsFromContext(ctx); ok {
		if p.TenantID != claims.TenantID {
			return nil, status.Error(codes.NotFound, "payment not found")
		}
	}
	return &paymentv1.GetPaymentResponse{Payment: domainToProto(p)}, nil
}

// GetPaymentByOrder retrieves the payment for an order.
func (h *PaymentHandler) GetPaymentByOrder(ctx context.Context, req *paymentv1.GetPaymentByOrderRequest) (*paymentv1.GetPaymentByOrderResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	p, err := h.getByOrderH.Handle(ctx, orderID)
	if err != nil {
		return nil, mapPaymentError(err)
	}
	// Verify tenant ownership — return NotFound to avoid existence oracle.
	if claims, ok := claimsFromContext(ctx); ok {
		if p.TenantID != claims.TenantID {
			return nil, status.Error(codes.NotFound, "payment not found")
		}
	}
	return &paymentv1.GetPaymentByOrderResponse{Payment: domainToProto(p)}, nil
}

// claimsFromContext extracts verified JWT claims stored by the auth middleware.
// In dev mode (no auth middleware installed), claims are absent; the caller
// receives a nil *sharedauth.Claims and ok=false.
func claimsFromContext(ctx context.Context) (*sharedauth.Claims, bool) {
	return middleware.ClaimsFromContext(ctx)
}

// tenantIDFromContext returns the tenant UUID from JWT claims when available,
// or parses fallbackRaw in dev/test mode. Returns an error if both paths fail.
func tenantIDFromContext(ctx context.Context, fallbackRaw string) (uuid.UUID, error) {
	if claims, ok := claimsFromContext(ctx); ok {
		return claims.TenantID, nil
	}
	id, err := uuid.Parse(fallbackRaw)
	if err != nil {
		return uuid.Nil, errors.New("invalid tenant_id")
	}
	return id, nil
}

func domainToProto(p *payment.Payment) *paymentv1.Payment {
	return &paymentv1.Payment{
		Id:              p.ID.String(),
		TenantId:        p.TenantID.String(),
		OrderId:         p.OrderID.String(),
		Amount:          p.Amount,
		Status:          domainStatusToProto(p.Status),
		Method:          domainMethodToProto(p.Method),
		ExternalId:      p.ExternalID,
		SnapToken:       p.SnapToken,
		SnapRedirectUrl: p.SnapRedirectURL,
		IdempotencyKey:  p.IdempotencyKey,
		CreatedAt:       p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func domainStatusToProto(s payment.PaymentStatus) paymentv1.PaymentStatus {
	switch s {
	case payment.StatusPending:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case payment.StatusSuccess:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_SUCCESS
	case payment.StatusFailed:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case payment.StatusVoided:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_VOIDED
	default:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func domainMethodToProto(m payment.PaymentMethod) paymentv1.PaymentMethod {
	switch m {
	case payment.MethodQRIS:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_QRIS
	case payment.MethodBankTransfer:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER
	case payment.MethodCreditCard:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case payment.MethodCash:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CASH
	default:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

func protoMethodToDomain(m paymentv1.PaymentMethod) payment.PaymentMethod {
	switch m {
	case paymentv1.PaymentMethod_PAYMENT_METHOD_QRIS:
		return payment.MethodQRIS
	case paymentv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER:
		return payment.MethodBankTransfer
	case paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD:
		return payment.MethodCreditCard
	case paymentv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return payment.MethodCash
	default:
		return payment.MethodQRIS
	}
}

func mapPaymentError(err error) error {
	switch {
	case errors.Is(err, payment.ErrPaymentNotFound):
		return status.Error(codes.NotFound, "payment not found")
	case errors.Is(err, payment.ErrInvalidAmount):
		return status.Error(codes.InvalidArgument, "invalid amount")
	case errors.Is(err, payment.ErrInvalidStatusTransition):
		return status.Error(codes.FailedPrecondition, "invalid payment status transition")
	case errors.Is(err, payment.ErrDuplicateIdempotencyKey):
		return status.Error(codes.AlreadyExists, "duplicate idempotency key")
	case errors.Is(err, payment.ErrGatewayFailure):
		return status.Error(codes.Unavailable, "payment gateway unavailable")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
