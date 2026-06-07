package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// CreateProductInput is the command payload.
type CreateProductInput struct {
	TenantID       uuid.UUID
	CategoryID     *uuid.UUID
	SKU            string
	Name           string
	Description    string
	BasePrice      int64
	TaxType        product.TaxType
	Variants       []product.Variant
	AddonGroups    []product.AddonGroup
	IdempotencyKey string
}

// CreateProductHandler orchestrates product creation.
type CreateProductHandler struct {
	repo product.ProductRepository
}

// NewCreateProductHandler creates a handler.
func NewCreateProductHandler(repo product.ProductRepository) *CreateProductHandler {
	return &CreateProductHandler{repo: repo}
}

// Handle creates a new product. Validates SKU uniqueness first.
func (h *CreateProductHandler) Handle(ctx context.Context, in CreateProductInput) (*product.Product, error) {
	if in.SKU != "" {
		_, err := h.repo.FindBySKU(ctx, in.TenantID, in.SKU)
		if err == nil {
			return nil, fmt.Errorf("CreateProduct: %w", product.ErrSKUAlreadyExists)
		}
		if !errors.Is(err, product.ErrProductNotFound) {
			return nil, fmt.Errorf("CreateProduct FindBySKU: %w", err)
		}
	}

	p, err := product.NewProduct(in.TenantID, in.CategoryID, in.SKU, in.Name, in.Description, in.BasePrice, in.TaxType)
	if err != nil {
		return nil, fmt.Errorf("CreateProduct domain: %w", err)
	}

	for _, v := range in.Variants {
		if err := p.AddVariant(v); err != nil {
			return nil, fmt.Errorf("CreateProduct AddVariant: %w", err)
		}
	}
	for _, ag := range in.AddonGroups {
		if err := p.AddAddonGroup(ag); err != nil {
			return nil, fmt.Errorf("CreateProduct AddAddonGroup: %w", err)
		}
	}

	if err := h.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("CreateProduct save: %w", err)
	}
	return p, nil
}
