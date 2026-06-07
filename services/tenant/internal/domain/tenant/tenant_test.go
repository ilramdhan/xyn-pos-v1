package tenant_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

func TestNewTenant_Valid(t *testing.T) {
	ten, err := domain.NewTenant("My Shop", "my-shop", domain.PlanTierFree)
	require.NoError(t, err)
	assert.Equal(t, "my-shop", ten.Slug)
	assert.Equal(t, domain.StatusActive, ten.Status)
	assert.Equal(t, domain.PlanTierFree, ten.Plan.Tier)
	assert.Equal(t, 1, ten.Plan.MaxBranches)
}

func TestNewTenant_EmptyName_ReturnsErrInvalidTenantName(t *testing.T) {
	_, err := domain.NewTenant("  ", "valid-slug", domain.PlanTierFree)
	assert.ErrorIs(t, err, domain.ErrInvalidTenantName)
}

func TestNewTenant_InvalidSlug_ReturnsErrInvalidSlug(t *testing.T) {
	cases := []string{"UPPER", "with space", "-startdash", "enddash-", "a", ""}
	for _, slug := range cases {
		_, err := domain.NewTenant("Name", slug, domain.PlanTierFree)
		assert.ErrorIs(t, err, domain.ErrInvalidSlug, "slug=%q should be invalid", slug)
	}
}

func TestNewTenant_ValidSlugs(t *testing.T) {
	cases := []string{"my-shop", "ab", "shop123", "a1b2c3"}
	for _, slug := range cases {
		_, err := domain.NewTenant("Name", slug, domain.PlanTierFree)
		assert.NoError(t, err, "slug=%q should be valid", slug)
	}
}

func TestNewTenant_PopEvents_ReturnsTenantCreatedEvent(t *testing.T) {
	ten, err := domain.NewTenant("Shop", "shop-one", domain.PlanTierGrowth)
	require.NoError(t, err)
	evts := ten.PopEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "tenant.created", evts[0].EventType())
	// Second pop should be empty
	assert.Empty(t, ten.PopEvents())
}

func TestAddBranch_WithinLimit_Succeeds(t *testing.T) {
	ten, _ := domain.NewTenant("Shop", "shop-two", domain.PlanTierFree)
	ten.PopEvents() // clear creation event

	branch := domain.NewBranch(ten.ID, "Main Branch", domain.Address{City: "Jakarta"}, "Asia/Jakarta")
	err := ten.AddBranch(branch)
	require.NoError(t, err)
	assert.Len(t, ten.Branches, 1)

	evts := ten.PopEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "tenant.branch_added", evts[0].EventType())
}

func TestAddBranch_ExceedsFreePlanLimit_ReturnsErrBranchLimitReached(t *testing.T) {
	ten, _ := domain.NewTenant("Shop", "shop-three", domain.PlanTierFree)
	// Free plan: max 1 branch
	b1 := domain.NewBranch(ten.ID, "Branch 1", domain.Address{}, "Asia/Jakarta")
	require.NoError(t, ten.AddBranch(b1))

	b2 := domain.NewBranch(ten.ID, "Branch 2", domain.Address{}, "Asia/Jakarta")
	err := ten.AddBranch(b2)
	assert.ErrorIs(t, err, domain.ErrBranchLimitReached)
}

func TestPlanFromTier_Growth_Returns10MaxBranches(t *testing.T) {
	plan := domain.PlanFromTier(domain.PlanTierGrowth)
	assert.Equal(t, 10, plan.MaxBranches)
}

func TestPlanFromTier_Enterprise_Returns999MaxBranches(t *testing.T) {
	plan := domain.PlanFromTier(domain.PlanTierEnterprise)
	assert.Equal(t, 999, plan.MaxBranches)
}

func TestNewTenant_LongName_IsValid(t *testing.T) {
	longName := strings.Repeat("a", 200)
	ten, err := domain.NewTenant(longName, "long-name", domain.PlanTierFree)
	require.NoError(t, err)
	assert.Equal(t, longName, ten.Name)
}
