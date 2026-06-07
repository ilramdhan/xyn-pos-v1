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

// TenantStatus represents the lifecycle state of a tenant.
type TenantStatus string

const (
	StatusActive    TenantStatus = "active"
	StatusSuspended TenantStatus = "suspended"
	StatusDeleted   TenantStatus = "deleted"
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

// Tenant is the aggregate root for the Tenant bounded context.
type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      Plan
	Status    TenantStatus
	Branches  []Branch
	CreatedAt time.Time
	UpdatedAt time.Time

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
	t := &Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Plan:      PlanFromTier(tier),
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	t.events = append(t.events, TenantCreatedEvent{
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
