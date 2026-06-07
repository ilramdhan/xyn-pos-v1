package user

import "errors"

// Sentinel errors for the user domain.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailTaken         = errors.New("email already taken in this tenant")
	ErrInvalidPIN         = errors.New("invalid PIN: must be 4–6 digits")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrBranchAccessDenied = errors.New("branch access denied for this user")
	ErrInvalidEmail       = errors.New("email cannot be empty")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidFullName    = errors.New("full name cannot be empty")
)
