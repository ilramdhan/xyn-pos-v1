package tenant_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

func TestNewTenant_StartsOnTrial(t *testing.T) {
	ten, err := tenant.NewTenant("Warung Padang", "warung-padang", tenant.PlanTierFree)
	require.NoError(t, err)
	assert.Equal(t, tenant.SubscriptionTrial, ten.SubscriptionStatus)
	assert.NotNil(t, ten.TrialEndsAt)
}

func TestIsSubscriptionActive_Trial_BeforeExpiry(t *testing.T) {
	ten := newTrialTenant(t)
	assert.True(t, ten.IsSubscriptionActive(time.Now()))
}

func TestIsSubscriptionActive_Trial_AfterExpiry(t *testing.T) {
	ten := newTrialTenant(t)
	future := time.Now().Add(30 * 24 * time.Hour) // 30 days = past 14-day trial
	assert.False(t, ten.IsSubscriptionActive(future))
}

func TestIsSubscriptionActive_Active_AlwaysTrue(t *testing.T) {
	ten := newActiveTenant(t)
	assert.True(t, ten.IsSubscriptionActive(time.Now().Add(365*24*time.Hour)))
}

func TestCheckSubscriptionAccess_ExpiredTrial_ReturnsError(t *testing.T) {
	ten := newTrialTenant(t)
	future := time.Now().Add(30 * 24 * time.Hour)
	err := ten.CheckSubscriptionAccess(future)
	assert.ErrorIs(t, err, tenant.ErrSubscriptionExpired)
}

func TestUpgradePlan_FromTrialToGrowth(t *testing.T) {
	ten := newTrialTenant(t)

	err := ten.UpgradePlan(tenant.PlanTierGrowth)
	require.NoError(t, err)
	assert.Equal(t, tenant.PlanTierGrowth, ten.Plan.Tier)
	assert.Equal(t, tenant.SubscriptionActive, ten.SubscriptionStatus)
	assert.Nil(t, ten.TrialEndsAt)
}

func TestUpgradePlan_EmitsPlanUpgradedEvent(t *testing.T) {
	ten := newTrialTenant(t)
	_ = ten.PopEvents() // clear creation event

	err := ten.UpgradePlan(tenant.PlanTierGrowth)
	require.NoError(t, err)

	evs := ten.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(tenant.PlanUpgradedEvent)
	require.True(t, ok)
	assert.Equal(t, ten.ID, ev.TenantID)
	assert.Equal(t, tenant.PlanTierFree, ev.OldTier)
	assert.Equal(t, tenant.PlanTierGrowth, ev.NewTier)
	assert.False(t, ev.OccurredAt.IsZero())
	assert.Empty(t, ten.PopEvents()) // second pop must return nothing
}

func TestUpgradePlan_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current tenant.PlanTier
		next    tenant.PlanTier
		wantErr error
	}{
		{"enterprise to free", tenant.PlanTierEnterprise, tenant.PlanTierFree, tenant.ErrDowngradeNotAllowed},
		{"enterprise to growth", tenant.PlanTierEnterprise, tenant.PlanTierGrowth, tenant.ErrDowngradeNotAllowed},
		{"growth to free", tenant.PlanTierGrowth, tenant.PlanTierFree, tenant.ErrDowngradeNotAllowed},
		{"same tier free", tenant.PlanTierFree, tenant.PlanTierFree, tenant.ErrSameTierUpgrade},
		{"same tier growth", tenant.PlanTierGrowth, tenant.PlanTierGrowth, tenant.ErrSameTierUpgrade},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build a tenant already at the desired starting tier via NewTenant + upgrades.
			ten, err := tenant.NewTenant("Test", "test-slug", tenant.PlanTierFree)
			require.NoError(t, err)
			if tc.current == tenant.PlanTierGrowth || tc.current == tenant.PlanTierEnterprise {
				require.NoError(t, ten.UpgradePlan(tenant.PlanTierGrowth))
			}
			if tc.current == tenant.PlanTierEnterprise {
				require.NoError(t, ten.UpgradePlan(tenant.PlanTierEnterprise))
			}
			err = ten.UpgradePlan(tc.next)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCancelSubscription_Success(t *testing.T) {
	ten := newActiveTenant(t)

	err := ten.CancelSubscription()
	require.NoError(t, err)
	assert.Equal(t, tenant.SubscriptionCancelled, ten.SubscriptionStatus)
	assert.Nil(t, ten.TrialEndsAt)
}

func TestCancelSubscription_EmitsEvent(t *testing.T) {
	ten := newActiveTenant(t)
	_ = ten.PopEvents()

	_ = ten.CancelSubscription()
	evs := ten.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(tenant.SubscriptionCancelledEvent)
	require.True(t, ok)
	assert.Equal(t, ten.ID, ev.TenantID)
	assert.False(t, ev.OccurredAt.IsZero())
}

func TestCancelSubscription_AlreadyCancelled_ReturnsError(t *testing.T) {
	ten := newActiveTenant(t)
	require.NoError(t, ten.CancelSubscription())
	err := ten.CancelSubscription()
	assert.ErrorIs(t, err, tenant.ErrSubscriptionAlreadyCancelled)
}

// newTrialTenant creates a new tenant in trial state (via NewTenant).
func newTrialTenant(t *testing.T) *tenant.Tenant {
	t.Helper()
	ten, err := tenant.NewTenant("Test Tenant", "test-tenant", tenant.PlanTierFree)
	require.NoError(t, err)
	return ten
}

// newActiveTenant creates a tenant that has upgraded to enterprise (SubscriptionActive).
func newActiveTenant(t *testing.T) *tenant.Tenant {
	t.Helper()
	ten, err := tenant.NewTenant("Test Tenant", "test-tenant-active", tenant.PlanTierFree)
	require.NoError(t, err)
	require.NoError(t, ten.UpgradePlan(tenant.PlanTierGrowth))
	require.NoError(t, ten.UpgradePlan(tenant.PlanTierEnterprise))
	_ = ten.PopEvents() // clear creation + upgrade events
	return ten
}
