package stock

import "errors"

var (
	ErrStockLedgerNotFound = errors.New("stock ledger not found")
	ErrBOMRecipeNotFound   = errors.New("BOM recipe not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrInvalidQuantity     = errors.New("quantity must be non-zero")
	ErrInvalidDelta        = errors.New("adjustment delta must be non-zero")
)
