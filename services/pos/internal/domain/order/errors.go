package order

import "errors"

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderNotDraft           = errors.New("order must be in DRAFT status")
	ErrOrderEmpty              = errors.New("order must have at least one item")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrDiscountExceedsSubtotal = errors.New("fixed discount cannot exceed subtotal")
	ErrDiscountPercentInvalid  = errors.New("percent discount must be 0.01% – 100%")
	ErrItemNotFound            = errors.New("order item not found")
	ErrShiftNotFound           = errors.New("shift not found")
	ErrShiftAlreadyClosed      = errors.New("shift is already closed")
	ErrShiftAlreadyOpen        = errors.New("cashier already has an open shift at this branch")
)
