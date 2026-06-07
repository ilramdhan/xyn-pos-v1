package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Role represents the RBAC role of a user within a tenant.
type Role string

// Valid Role constants.
const (
	RoleOwner        Role = "owner"
	RoleManager      Role = "manager"
	RoleCashier      Role = "cashier"
	RoleKitchenStaff Role = "kitchen_staff"
)

// IsValid reports whether r is a recognised role value.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleManager, RoleCashier, RoleKitchenStaff:
		return true
	}
	return false
}

// User is the aggregate root for the user bounded context.
type User struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	KeycloakID  string
	Email       string
	FullName    string
	Role        Role
	BranchScope []uuid.UUID
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	events      []DomainEvent
}

// NewUser constructs and validates a new User aggregate.
func NewUser(tenantID uuid.UUID, keycloakID, email, fullName string, role Role, branchScope []uuid.UUID) (*User, error) {
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if fullName == "" {
		return nil, ErrInvalidFullName
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRole, role)
	}

	now := time.Now().UTC()
	u := &User{
		ID:          uuid.New(),
		TenantID:    tenantID,
		KeycloakID:  keycloakID,
		Email:       email,
		FullName:    fullName,
		Role:        role,
		BranchScope: branchScope,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	u.events = append(u.events, UserRegisteredEvent{
		UserID:   u.ID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
	})
	return u, nil
}

// Deactivate marks the user as inactive. Returns ErrUserInactive if already inactive.
func (u *User) Deactivate() error {
	if !u.IsActive {
		return ErrUserInactive
	}
	u.IsActive = false
	u.UpdatedAt = time.Now().UTC()
	u.events = append(u.events, UserDeactivatedEvent{UserID: u.ID, TenantID: u.TenantID})
	return nil
}

// CanAccessBranch returns true when the user has access to branchID.
// Users with an empty BranchScope have access to all branches.
func (u *User) CanAccessBranch(branchID uuid.UUID) bool {
	if len(u.BranchScope) == 0 {
		return true
	}
	for _, id := range u.BranchScope {
		if id == branchID {
			return true
		}
	}
	return false
}

// PopEvents returns accumulated domain events and clears the internal slice.
func (u *User) PopEvents() []DomainEvent {
	evs := u.events
	u.events = nil
	return evs
}
