package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// CreateCategoryInput is the command payload.
type CreateCategoryInput struct {
	TenantID  uuid.UUID
	Name      string
	SortOrder int
	ParentID  *uuid.UUID
}

// CreateCategoryHandler creates a product category.
type CreateCategoryHandler struct {
	repo product.CategoryRepository
}

// NewCreateCategoryHandler creates a handler.
func NewCreateCategoryHandler(repo product.CategoryRepository) *CreateCategoryHandler {
	return &CreateCategoryHandler{repo: repo}
}

// Handle creates and persists a category.
func (h *CreateCategoryHandler) Handle(ctx context.Context, in CreateCategoryInput) (*product.Category, error) {
	now := time.Now().UTC()
	c := &product.Category{
		ID:        uuid.New(),
		TenantID:  in.TenantID,
		Name:      in.Name,
		SortOrder: in.SortOrder,
		ParentID:  in.ParentID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("CreateCategory save: %w", err)
	}
	return c, nil
}
