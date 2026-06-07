package stock

import (
	"github.com/google/uuid"
)

// BOMRecipe is one line in a Bill of Materials.
// A product may have multiple BOMRecipes (one per ingredient).
type BOMRecipe struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	ProductID           uuid.UUID
	IngredientProductID uuid.UUID // the ingredient being consumed
	QuantityPerUnit     int64     // how much of the ingredient per 1 unit sold
	Unit                string    // "g", "ml", "pcs"
}

// NewBOMRecipe creates a BOM line item.
func NewBOMRecipe(tenantID, productID, ingredientID uuid.UUID, qtyPerUnit int64, unit string) (*BOMRecipe, error) {
	if qtyPerUnit <= 0 {
		return nil, ErrInvalidQuantity
	}
	return &BOMRecipe{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		ProductID:           productID,
		IngredientProductID: ingredientID,
		QuantityPerUnit:     qtyPerUnit,
		Unit:                unit,
	}, nil
}
