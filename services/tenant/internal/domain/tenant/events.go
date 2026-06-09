package tenant

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the base interface for all tenant domain events.
type DomainEvent interface {
	EventType() string
}

// CreatedEvent is fired when a new tenant is registered.
type CreatedEvent struct {
	TenantID  uuid.UUID
	Name      string
	Slug      string
	Plan      PlanTier
	CreatedAt time.Time
}

// EventType returns the event type identifier.
func (e CreatedEvent) EventType() string { return "tenant.created" }

// BranchAddedEvent is fired when a branch is added to a tenant.
type BranchAddedEvent struct {
	TenantID  uuid.UUID
	BranchID  uuid.UUID
	Name      string
	CreatedAt time.Time
}

// EventType returns the event type identifier.
func (e BranchAddedEvent) EventType() string { return "tenant.branch_added" }

// PlanUpgradedEvent is fired when a tenant upgrades their plan.
type PlanUpgradedEvent struct {
	TenantID   uuid.UUID
	OldTier    PlanTier
	NewTier    PlanTier
	OccurredAt time.Time
}

func (e PlanUpgradedEvent) EventType() string { return "tenant.plan_upgraded" }

// SubscriptionCancelledEvent is fired when a tenant cancels their subscription.
type SubscriptionCancelledEvent struct {
	TenantID   uuid.UUID
	OccurredAt time.Time
}

func (e SubscriptionCancelledEvent) EventType() string { return "tenant.subscription_cancelled" }
