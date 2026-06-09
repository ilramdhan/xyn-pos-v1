package tenant

import "errors"

// Sentinel errors for the tenant domain.
var (
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrSlugAlreadyTaken   = errors.New("slug already taken")
	ErrBranchLimitReached = errors.New("branch limit reached for plan")
	ErrInvalidTenantName  = errors.New("tenant name cannot be empty")
	ErrInvalidSlug        = errors.New("slug must be lowercase alphanumeric with hyphens")

	ErrSubscriptionExpired          = errors.New("subscription has expired or been cancelled")
	ErrDowngradeNotAllowed          = errors.New("plan downgrade is not allowed")
	ErrSameTierUpgrade              = errors.New("tenant is already on this plan tier")
	ErrSubscriptionAlreadyCancelled = errors.New("subscription is already cancelled")
)
