package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// CreateBranchCommand carries the input for adding a branch to a tenant.
type CreateBranchCommand struct {
	TenantID uuid.UUID
	Name     string
	Address  domain.Address
	Timezone string
}

// CreateBranchHandler handles the CreateBranch command.
type CreateBranchHandler struct {
	repo domain.Repository
}

// NewCreateBranchHandler constructs a CreateBranchHandler.
func NewCreateBranchHandler(repo domain.Repository) *CreateBranchHandler {
	return &CreateBranchHandler{repo: repo}
}

// Handle executes the CreateBranch command, enforcing plan branch limits.
func (h *CreateBranchHandler) Handle(ctx context.Context, cmd CreateBranchCommand) (*domain.Branch, error) {
	t, err := h.repo.FindByID(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.FindByID: %w", err)
	}

	// Guard: tenant must have an active subscription to add branches.
	if err := t.CheckSubscriptionAccess(time.Now()); err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.CheckSubscriptionAccess: %w", err)
	}

	tz := cmd.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	branch := domain.NewBranch(t.ID, cmd.Name, cmd.Address, tz)
	if err := t.AddBranch(branch); err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.AddBranch: %w", err)
	}

	if err := h.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.Save: %w", err)
	}

	return &branch, nil
}
