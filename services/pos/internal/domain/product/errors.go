package product

import "errors"

var (
	ErrProductNotFound        = errors.New("product not found")
	ErrSKUAlreadyExists       = errors.New("SKU already exists in this tenant")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrProductHasActiveOrders = errors.New("cannot archive product with active orders")
	ErrVariantPriceInvalid    = errors.New("variant final price (base + delta) must be >= 0")
	ErrInvalidPrice           = errors.New("base price must be >= 0")
	ErrInvalidName            = errors.New("product name cannot be empty")
	ErrInvalidAddonGroup      = errors.New("addon group max_selections must be >= 1")
	ErrProductAlreadyArchived = errors.New("product is already archived")
)
