package payment

import "errors"

var (
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrInvalidAmount           = errors.New("payment amount must be > 0")
	ErrInvalidStatusTransition = errors.New("invalid payment status transition")
	ErrDuplicateIdempotencyKey = errors.New("payment with this idempotency key already exists")
	ErrGatewayFailure          = errors.New("payment gateway failure")
	ErrReceiptNotAllowed       = errors.New("receipt can only be generated for a successful payment")
	ErrReceiptNotFound         = errors.New("receipt not found")
	ErrPartialRefundInvalid    = errors.New("refund amount must be > 0 and ≤ original payment amount")
)
