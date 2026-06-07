package payment

import "errors"

var (
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrInvalidAmount           = errors.New("payment amount must be > 0")
	ErrInvalidStatusTransition = errors.New("invalid payment status transition")
	ErrDuplicateIdempotencyKey = errors.New("payment with this idempotency key already exists")
	ErrGatewayFailure          = errors.New("payment gateway failure")
)
