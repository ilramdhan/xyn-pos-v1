package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// UpdatePlanCommand carries the input for upgrading a tenant's plan.
type UpdatePlanCommand struct {
	TenantID uuid.UUID
	NewTier  domain.PlanTier
}

// UpdatePlanHandler handles the UpdatePlan command.
type UpdatePlanHandler struct {
	repo domain.Repository
}

// NewUpdatePlanHandler constructs an UpdatePlanHandler.
func NewUpdatePlanHandler(repo domain.Repository) *UpdatePlanHandler {
	return &UpdatePlanHandler{repo: repo}
}

// Handle upgrades the tenant's plan tier.
func (h *UpdatePlanHandler) Handle(ctx context.Context, cmd UpdatePlanCommand) error {
	t, err := h.repo.FindByID(ctx, cmd.TenantID)
	if err != nil {
		return fmt.Errorf("UpdatePlanHandler.FindByID: %w", err)
	}

	if err := t.UpgradePlan(cmd.NewTier); err != nil {
		return fmt.Errorf("UpdatePlanHandler.UpgradePlan: %w", err)
	}

	if err := h.repo.Save(ctx, t); err != nil {
		return fmt.Errorf("UpdatePlanHandler.Save: %w", err)
	}

	return nil
}
