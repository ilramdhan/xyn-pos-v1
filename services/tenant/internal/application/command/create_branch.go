package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type CreateBranchCommand struct {
	IdempotencyKey string
	TenantID       uuid.UUID
	Name           string
	Address        domain.Address
	Timezone       string
}

type CreateBranchHandler struct {
	repo domain.Repository
}

func NewCreateBranchHandler(repo domain.Repository) *CreateBranchHandler {
	return &CreateBranchHandler{repo: repo}
}

func (h *CreateBranchHandler) Handle(ctx context.Context, cmd CreateBranchCommand) (*domain.Branch, error) {
	t, err := h.repo.FindByID(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("CreateBranchHandler.FindByID: %w", err)
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
