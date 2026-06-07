package user

import "github.com/google/uuid"

// DomainEvent is the marker interface for all user domain events.
type DomainEvent interface{ userEvent() }

// UserRegisteredEvent is emitted when a new user is created.
type UserRegisteredEvent struct { //nolint:revive // package-qualified name is intentional for clarity
	UserID   uuid.UUID
	TenantID uuid.UUID
	Email    string
	Role     Role
}

func (UserRegisteredEvent) userEvent() {}

// UserDeactivatedEvent is emitted when a user is deactivated.
type UserDeactivatedEvent struct { //nolint:revive // package-qualified name is intentional for clarity
	UserID   uuid.UUID
	TenantID uuid.UUID
}

func (UserDeactivatedEvent) userEvent() {}
