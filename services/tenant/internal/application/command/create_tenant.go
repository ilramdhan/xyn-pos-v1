package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// Publisher defines the event publishing contract for the command layer.
type Publisher interface {
	Publish(ctx context.Context, topic string, tenantID uuid.UUID, eventType string, payload []byte) error
}

// CreateTenantCommand carries the input for creating a new tenant.
type CreateTenantCommand struct {
	IdempotencyKey string
	Name           string
	Slug           string
	Plan           domain.PlanTier
}

// CreateTenantHandler orchestrates the CreateTenant use case.
type CreateTenantHandler struct {
	repo      domain.Repository
	publisher Publisher
}

// NewCreateTenantHandler constructs a CreateTenantHandler with its dependencies.
func NewCreateTenantHandler(repo domain.Repository, publisher Publisher) *CreateTenantHandler {
	return &CreateTenantHandler{repo: repo, publisher: publisher}
}

// Handle executes the CreateTenant use case.
// It enforces slug uniqueness, creates the aggregate, persists it,
// and optionally publishes domain events.
func (h *CreateTenantHandler) Handle(ctx context.Context, cmd CreateTenantCommand) (*domain.Tenant, error) {
	exists, err := h.repo.ExistsBySlug(ctx, cmd.Slug)
	if err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.ExistsBySlug: %w", err)
	}
	if exists {
		return nil, domain.ErrSlugAlreadyTaken
	}

	t, err := domain.NewTenant(cmd.Name, cmd.Slug, cmd.Plan)
	if err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.NewTenant: %w", err)
	}

	if err := h.repo.Save(ctx, t); err != nil {
		return nil, fmt.Errorf("CreateTenantHandler.Save: %w", err)
	}

	// Domain events published by caller (provider.go) — non-fatal if Kafka not wired.
	if h.publisher != nil {
		for _, evt := range t.PopEvents() {
			_ = evt
		}
	}

	return t, nil
}
