package tenant

import "errors"

var (
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrSlugAlreadyTaken   = errors.New("slug already taken")
	ErrBranchLimitReached = errors.New("branch limit reached for plan")
	ErrInvalidTenantName  = errors.New("tenant name cannot be empty")
	ErrInvalidSlug        = errors.New("slug must be lowercase alphanumeric with hyphens")
)
