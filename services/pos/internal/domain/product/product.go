package product

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaxType determines how tax is applied.
type TaxType string

const (
	TaxTypePPN  TaxType = "ppn" // 11% exclusive
	TaxTypePB1  TaxType = "pb1" // 10% inclusive (reporting)
	TaxTypeNone TaxType = "none"
)

// IsValid reports whether t is a recognised TaxType value.
func (t TaxType) IsValid() bool {
	switch t {
	case TaxTypePPN, TaxTypePB1, TaxTypeNone:
		return true
	}
	return false
}

// Category groups products for display. Tenant-global — all branches see the same catalog.
type Category struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	SortOrder int
	ParentID  *uuid.UUID
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Variant is a product option with a price delta from the base price.
type Variant struct {
	ID         uuid.UUID
	ProductID  uuid.UUID
	Name       string
	PriceDelta int64 // may be negative (discount variant); FinalPrice = Product.BasePrice + PriceDelta
	SKU        string
	IsActive   bool
}

// FinalPrice returns the effective variant price. Returns 0 if negative.
func (v Variant) FinalPrice(basePrice int64) int64 {
	fp := basePrice + v.PriceDelta
	if fp < 0 {
		return 0
	}
	return fp
}

// Addon is a selectable extra within an addon group.
type Addon struct {
	ID           uuid.UUID
	AddonGroupID uuid.UUID
	Name         string
	Price        int64 // sen, >= 0
	IsActive     bool
}

// AddonGroup is a set of add-ons attached to a product.
type AddonGroup struct {
	ID            uuid.UUID
	ProductID     uuid.UUID
	Name          string
	IsRequired    bool
	MaxSelections int
	Addons        []Addon
}

// BranchProduct holds a branch-specific price override for a product.
type BranchProduct struct {
	BranchID      uuid.UUID
	ProductID     uuid.UUID
	OverridePrice *int64 // nil = use product.BasePrice
	IsAvailable   bool
	UpdatedAt     time.Time
}

// Product is the catalog aggregate root.
type Product struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CategoryID  *uuid.UUID
	SKU         string
	Name        string
	Description string
	BasePrice   int64 // minor units (sen). Rp 25.000 = 2500000
	TaxType     TaxType
	Variants    []Variant
	AddonGroups []AddonGroup
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProduct constructs a Product aggregate. Validates name and base price.
func NewProduct(tenantID uuid.UUID, categoryID *uuid.UUID, sku, name, description string, basePrice int64, taxType TaxType) (*Product, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if basePrice < 0 {
		return nil, ErrInvalidPrice
	}
	if !taxType.IsValid() {
		return nil, ErrInvalidTaxType
	}

	now := time.Now().UTC()
	return &Product{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CategoryID:  categoryID,
		SKU:         sku,
		Name:        name,
		Description: description,
		BasePrice:   basePrice,
		TaxType:     taxType,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// AddVariant appends a variant. Returns ErrVariantPriceInvalid if final price < 0.
func (p *Product) AddVariant(v Variant) error {
	if p.BasePrice+v.PriceDelta < 0 {
		return fmt.Errorf("AddVariant %q: %w", v.Name, ErrVariantPriceInvalid)
	}
	v.ProductID = p.ID
	p.Variants = append(p.Variants, v)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// AddAddonGroup appends an addon group. MaxSelections must be >= 1.
func (p *Product) AddAddonGroup(ag AddonGroup) error {
	if ag.MaxSelections < 1 {
		return fmt.Errorf("AddAddonGroup %q: %w", ag.Name, ErrInvalidAddonGroup)
	}
	ag.ProductID = p.ID
	p.AddonGroups = append(p.AddonGroups, ag)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Archive soft-deletes the product. Returns ErrProductAlreadyArchived if already inactive.
func (p *Product) Archive() error {
	if !p.IsActive {
		return ErrProductAlreadyArchived
	}
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
	return nil
}
