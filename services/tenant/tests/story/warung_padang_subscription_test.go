// Package story_test — Phase 4 Story C: Subscription Trial Lifecycle.
//
// Covers:
//   - NewTenant starts in SubscriptionTrial with a 14-day trial window
//   - CheckSubscriptionAccess allows access during trial
//   - Expired trial blocks access
//   - UpgradePlan: valid and invalid transitions
//   - CancelSubscription
//
// Run with: go test ./services/tenant/tests/story/... -v
package story_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenantdomain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// TestStoryC_SubscriptionLifecycle validates the full subscription trial lifecycle.
func TestStoryC_SubscriptionLifecycle(t *testing.T) {
	// =========================================================================
	// Step 1: New tenant starts in trial
	// =========================================================================
	t.Log("Step 1: New tenant created in trial status")

	tenant, err := tenantdomain.NewTenant("Warung Padang Sejati", "warung-padang-sejati", tenantdomain.PlanTierFree)
	require.NoError(t, err)
	_ = tenant.PopEvents() // clear TenantCreatedEvent
	assert.Equal(t, tenantdomain.SubscriptionTrial, tenant.SubscriptionStatus)
	require.NotNil(t, tenant.TrialEndsAt, "TrialEndsAt must be set on new tenant")
	assert.True(t, tenant.TrialEndsAt.After(time.Now()), "trial must end in the future")
	t.Logf("  Trial ends at: %s", tenant.TrialEndsAt.Format(time.RFC3339))

	// =========================================================================
	// Step 2: Access allowed during active trial
	// =========================================================================
	t.Log("Step 2: CheckSubscriptionAccess — allowed during trial window")

	now := time.Now()
	err = tenant.CheckSubscriptionAccess(now)
	require.NoError(t, err, "access must be allowed during active trial")

	assert.True(t, tenant.IsSubscriptionActive(now))

	// =========================================================================
	// Step 3: Access denied after trial expiry
	// =========================================================================
	t.Log("Step 3: CheckSubscriptionAccess — denied after trial expires")

	afterExpiry := tenant.TrialEndsAt.Add(time.Hour)
	err = tenant.CheckSubscriptionAccess(afterExpiry)
	assert.ErrorIs(t, err, tenantdomain.ErrSubscriptionExpired)

	assert.False(t, tenant.IsSubscriptionActive(afterExpiry))

	// =========================================================================
	// Step 4: Upgrade plan Free → Growth — trial becomes active
	// =========================================================================
	t.Log("Step 4: Upgrade plan Free → Growth")

	err = tenant.UpgradePlan(tenantdomain.PlanTierGrowth)
	require.NoError(t, err)
	assert.Equal(t, tenantdomain.PlanTierGrowth, tenant.Plan.Tier)
	assert.Equal(t, tenantdomain.SubscriptionActive, tenant.SubscriptionStatus)

	// After upgrade, access allowed regardless of trial window.
	err = tenant.CheckSubscriptionAccess(afterExpiry)
	require.NoError(t, err, "active subscription must allow access after upgrade")

	// Verify PlanUpgradedEvent emitted.
	evs := tenant.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(tenantdomain.PlanUpgradedEvent)
	require.True(t, ok, "expected PlanUpgradedEvent")
	assert.Equal(t, tenantdomain.PlanTierFree, ev.OldTier)
	assert.Equal(t, tenantdomain.PlanTierGrowth, ev.NewTier)
	assert.False(t, ev.OccurredAt.IsZero())

	// =========================================================================
	// Step 5: Upgrade Growth → Enterprise
	// =========================================================================
	t.Log("Step 5: Upgrade plan Growth → Enterprise")

	err = tenant.UpgradePlan(tenantdomain.PlanTierEnterprise)
	require.NoError(t, err)
	assert.Equal(t, tenantdomain.PlanTierEnterprise, tenant.Plan.Tier)
	_ = tenant.PopEvents()

	// =========================================================================
	// Step 6: Invalid upgrades rejected
	// =========================================================================
	t.Log("Step 6: Guards — invalid plan transitions rejected")

	// Same-tier
	err = tenant.UpgradePlan(tenantdomain.PlanTierEnterprise)
	assert.ErrorIs(t, err, tenantdomain.ErrSameTierUpgrade)

	// Downgrade
	err = tenant.UpgradePlan(tenantdomain.PlanTierGrowth)
	assert.ErrorIs(t, err, tenantdomain.ErrDowngradeNotAllowed)

	err = tenant.UpgradePlan(tenantdomain.PlanTierFree)
	assert.ErrorIs(t, err, tenantdomain.ErrDowngradeNotAllowed)

	// =========================================================================
	// Step 7: Cancel subscription
	// =========================================================================
	t.Log("Step 7: Cancel subscription")

	err = tenant.CancelSubscription()
	require.NoError(t, err)
	assert.Equal(t, tenantdomain.SubscriptionCancelled, tenant.SubscriptionStatus)

	evs = tenant.PopEvents()
	require.Len(t, evs, 1)
	_, ok = evs[0].(tenantdomain.SubscriptionCancelledEvent)
	require.True(t, ok, "expected SubscriptionCancelledEvent")

	// Double-cancel rejected.
	err = tenant.CancelSubscription()
	assert.ErrorIs(t, err, tenantdomain.ErrSubscriptionAlreadyCancelled)
}
