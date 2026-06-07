package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// UpdateProductInput is the command payload.
type UpdateProductInput struct {
	ProductID      uuid.UUID
	Name           string
	Description    string
	BasePrice      int64
	TaxType        product.TaxType
	CategoryID     *uuid.UUID
	IdempotencyKey string
}

// UpdateProductHandler updates a product's core fields.
type UpdateProductHandler struct {
	repo product.ProductRepository
}

// NewUpdateProductHandler creates a handler.
func NewUpdateProductHandler(repo product.ProductRepository) *UpdateProductHandler {
	return &UpdateProductHandler{repo: repo}
}

// Handle updates a product's mutable fields.
func (h *UpdateProductHandler) Handle(ctx context.Context, in UpdateProductInput) (*product.Product, error) {
	p, err := h.repo.FindByID(ctx, in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("UpdateProduct FindByID: %w", err)
	}
	if in.Name != "" {
		p.Name = in.Name
	}
	if in.Description != "" {
		p.Description = in.Description
	}
	if in.BasePrice >= 0 {
		p.BasePrice = in.BasePrice
	}
	if in.TaxType != "" && in.TaxType.IsValid() {
		p.TaxType = in.TaxType
	}
	if in.CategoryID != nil {
		p.CategoryID = in.CategoryID
	}
	p.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("UpdateProduct update: %w", err)
	}
	return p, nil
}
