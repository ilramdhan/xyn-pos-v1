package tenant

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the base interface for all tenant domain events.
type DomainEvent interface {
	EventType() string
}

// TenantCreatedEvent is fired when a new tenant is registered.
type TenantCreatedEvent struct {
	TenantID  uuid.UUID
	Name      string
	Slug      string
	Plan      PlanTier
	CreatedAt time.Time
}

func (e TenantCreatedEvent) EventType() string { return "tenant.created" }

// BranchAddedEvent is fired when a branch is added to a tenant.
type BranchAddedEvent struct {
	TenantID  uuid.UUID
	BranchID  uuid.UUID
	Name      string
	CreatedAt time.Time
}

func (e BranchAddedEvent) EventType() string { return "tenant.branch_added" }
