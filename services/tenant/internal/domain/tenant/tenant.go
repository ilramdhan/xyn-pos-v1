package tenant

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PlanTier defines subscription plan levels.
type PlanTier string

// Subscription plan tier values.
const (
	PlanTierFree       PlanTier = "free"
	PlanTierGrowth     PlanTier = "growth"
	PlanTierEnterprise PlanTier = "enterprise"
)

// Plan holds the tier and its limits.
type Plan struct {
	Tier        PlanTier
	MaxBranches int
}

// PlanFromTier returns the Plan for a given tier.
func PlanFromTier(tier PlanTier) Plan {
	switch tier {
	case PlanTierGrowth:
		return Plan{Tier: PlanTierGrowth, MaxBranches: 10}
	case PlanTierEnterprise:
		return Plan{Tier: PlanTierEnterprise, MaxBranches: 999}
	default:
		return Plan{Tier: PlanTierFree, MaxBranches: 1}
	}
}

// Status represents the lifecycle state of a tenant.
type Status string

// Tenant lifecycle status values.
const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// Address is a value object for physical locations.
type Address struct {
	Street     string
	City       string
	Province   string
	PostalCode string
	Country    string
}

// Branch represents a physical business location within a tenant.
type Branch struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Address   Address
	Timezone  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewBranch creates a new Branch value object.
func NewBranch(tenantID uuid.UUID, name string, address Address, timezone string) Branch {
	now := time.Now().UTC()
	return Branch{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Address:   address,
		Timezone:  timezone,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// tierRank defines the ordering of subscription tiers for upgrade/downgrade validation.
var tierRank = map[PlanTier]int{
	PlanTierFree:       0,
	PlanTierGrowth:     1,
	PlanTierEnterprise: 2,
}

// SubscriptionStatus is the state of the tenant's subscription.
type SubscriptionStatus string

// Subscription status values.
const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionTrial     SubscriptionStatus = "trial"
	SubscriptionExpired   SubscriptionStatus = "expired"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
)

// Tenant is the aggregate root for the Tenant bounded context.
type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      Plan
	Status    Status
	Branches  []Branch
	CreatedAt time.Time
	UpdatedAt time.Time

	SubscriptionStatus SubscriptionStatus
	TrialEndsAt        *time.Time // nil if not on trial

	events []DomainEvent
}

// NewTenant creates a new Tenant aggregate, validating invariants.
func NewTenant(name, slug string, tier PlanTier) (*Tenant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidTenantName
	}
	if !slugRegex.MatchString(slug) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidSlug, slug)
	}

	now := time.Now().UTC()
	trialEnd := now.Add(14 * 24 * time.Hour)
	t := &Tenant{
		ID:                 uuid.New(),
		Name:               name,
		Slug:               slug,
		Plan:               PlanFromTier(tier),
		Status:             StatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
		SubscriptionStatus: SubscriptionTrial,
		TrialEndsAt:        &trialEnd,
	}
	t.events = append(t.events, CreatedEvent{
		TenantID:  t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Plan:      t.Plan.Tier,
		CreatedAt: t.CreatedAt,
	})
	return t, nil
}

// AddBranch adds a branch, enforcing plan limits.
func (t *Tenant) AddBranch(branch Branch) error {
	if len(t.Branches) >= t.Plan.MaxBranches {
		return fmt.Errorf("%w: plan=%s max=%d", ErrBranchLimitReached, t.Plan.Tier, t.Plan.MaxBranches)
	}
	t.Branches = append(t.Branches, branch)
	t.UpdatedAt = time.Now().UTC()
	t.events = append(t.events, BranchAddedEvent{
		TenantID:  t.ID,
		BranchID:  branch.ID,
		Name:      branch.Name,
		CreatedAt: branch.CreatedAt,
	})
	return nil
}

// PopEvents returns and clears the tenant's uncommitted domain events.
func (t *Tenant) PopEvents() []DomainEvent {
	evts := t.events
	t.events = nil
	return evts
}

// IsSubscriptionActive returns true if the tenant can use the system.
// Trial tenants are active until TrialEndsAt; active subscriptions are always active.
func (t *Tenant) IsSubscriptionActive(now time.Time) bool {
	switch t.SubscriptionStatus {
	case SubscriptionActive:
		return true
	case SubscriptionTrial:
		return t.TrialEndsAt != nil && now.Before(*t.TrialEndsAt)
	default:
		return false
	}
}

// CheckSubscriptionAccess returns ErrSubscriptionExpired if the tenant cannot access the system.
func (t *Tenant) CheckSubscriptionAccess(now time.Time) error {
	if !t.IsSubscriptionActive(now) {
		return ErrSubscriptionExpired
	}
	return nil
}

// UpgradePlan upgrades the tenant to a new plan tier and activates the subscription.
// Returns ErrSameTierUpgrade if the new tier equals the current tier.
// Returns ErrDowngradeNotAllowed if the new tier is lower than the current tier.
func (t *Tenant) UpgradePlan(newTier PlanTier) error {
	if newTier == t.Plan.Tier {
		return ErrSameTierUpgrade
	}
	if tierRank[newTier] < tierRank[t.Plan.Tier] {
		return ErrDowngradeNotAllowed
	}
	newPlan := PlanFromTier(newTier)
	oldTier := t.Plan.Tier
	t.Plan = newPlan
	t.SubscriptionStatus = SubscriptionActive
	t.TrialEndsAt = nil
	t.UpdatedAt = time.Now().UTC()
	t.events = append(t.events, PlanUpgradedEvent{
		TenantID:   t.ID,
		OldTier:    oldTier,
		NewTier:    newTier,
		OccurredAt: t.UpdatedAt,
	})
	return nil
}

// CancelSubscription marks the subscription as cancelled.
// Returns ErrSubscriptionAlreadyCancelled if already cancelled.
func (t *Tenant) CancelSubscription() error {
	if t.SubscriptionStatus == SubscriptionCancelled {
		return ErrSubscriptionAlreadyCancelled
	}
	t.SubscriptionStatus = SubscriptionCancelled
	t.TrialEndsAt = nil
	t.UpdatedAt = time.Now().UTC()
	t.events = append(t.events, SubscriptionCancelledEvent{
		TenantID:   t.ID,
		OccurredAt: t.UpdatedAt,
	})
	return nil
}
