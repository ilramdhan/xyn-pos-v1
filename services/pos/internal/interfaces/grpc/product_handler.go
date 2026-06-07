package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	posv1 "github.com/xyn-pos/gen/pos/v1"
	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/application/query"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ProductHandler implements posv1.ProductServiceServer.
type ProductHandler struct {
	posv1.UnimplementedProductServiceServer
	createProductH   *command.CreateProductHandler
	updateProductH   *command.UpdateProductHandler
	archiveProductH  *command.ArchiveProductHandler
	createCategoryH  *command.CreateCategoryHandler
	reorderCategoryH *command.ReorderCategoriesHandler
	setBranchPriceH  *command.SetBranchPriceHandler
	getProductH      *query.GetProductHandler
	listProductsH    *query.ListProductsHandler
	lookupBySKUH     *query.LookupBySKUHandler
}

// NewProductHandler assembles the handler.
func NewProductHandler(
	createH *command.CreateProductHandler,
	updateH *command.UpdateProductHandler,
	archiveH *command.ArchiveProductHandler,
	createCatH *command.CreateCategoryHandler,
	reorderCatH *command.ReorderCategoriesHandler,
	setBranchH *command.SetBranchPriceHandler,
	getH *query.GetProductHandler,
	listH *query.ListProductsHandler,
	lookupH *query.LookupBySKUHandler,
) *ProductHandler {
	return &ProductHandler{
		createProductH:   createH,
		updateProductH:   updateH,
		archiveProductH:  archiveH,
		createCategoryH:  createCatH,
		reorderCategoryH: reorderCatH,
		setBranchPriceH:  setBranchH,
		getProductH:      getH,
		listProductsH:    listH,
		lookupBySKUH:     lookupH,
	}
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *posv1.CreateProductRequest) (*posv1.CreateProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	p, err := h.createProductH.Handle(ctx, command.CreateProductInput{
		TenantID:       tenantID,
		SKU:            req.Sku,
		Name:           req.Name,
		Description:    req.Description,
		BasePrice:      req.BasePrice,
		TaxType:        protoTaxTypeToDomain(req.TaxType),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.CreateProductResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, req *posv1.UpdateProductRequest) (*posv1.UpdateProductResponse, error) {
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	p, err := h.updateProductH.Handle(ctx, command.UpdateProductInput{
		ProductID:      productID,
		Name:           req.Name,
		Description:    req.Description,
		BasePrice:      req.BasePrice,
		TaxType:        protoTaxTypeToDomain(req.TaxType),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.UpdateProductResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) ArchiveProduct(ctx context.Context, req *posv1.ArchiveProductRequest) (*posv1.ArchiveProductResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	if err := h.archiveProductH.Handle(ctx, id); err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.ArchiveProductResponse{}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *posv1.GetProductRequest) (*posv1.GetProductResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	p, err := h.getProductH.Handle(ctx, id)
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.GetProductResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) ListProducts(ctx context.Context, req *posv1.ListProductsRequest) (*posv1.ListProductsResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	filter := product.Filter{
		ActiveOnly: req.ActiveOnly,
		Limit:      int(req.Pagination.GetLimit()),
		Offset:     0,
	}
	if req.CategoryId != "" {
		catID, err := uuid.Parse(req.CategoryId)
		if err == nil {
			filter.CategoryID = &catID
		}
	}
	result, err := h.listProductsH.Handle(ctx, tenantID, filter)
	if err != nil {
		return nil, mapProductError(err)
	}
	protoProducts := make([]*posv1.Product, len(result.Products))
	for i, p := range result.Products {
		protoProducts[i] = domainProductToProto(p)
	}
	return &posv1.ListProductsResponse{Products: protoProducts}, nil
}

func (h *ProductHandler) LookupBySKU(ctx context.Context, req *posv1.LookupBySKURequest) (*posv1.LookupBySKUResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	p, err := h.lookupBySKUH.Handle(ctx, tenantID, req.Sku)
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.LookupBySKUResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) CreateCategory(ctx context.Context, req *posv1.CreateCategoryRequest) (*posv1.CreateCategoryResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	c, err := h.createCategoryH.Handle(ctx, command.CreateCategoryInput{
		TenantID:  tenantID,
		Name:      req.Name,
		SortOrder: int(req.SortOrder),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create category")
	}
	return &posv1.CreateCategoryResponse{Category: domainCategoryToProto(c)}, nil
}

func (h *ProductHandler) ReorderCategories(ctx context.Context, req *posv1.ReorderCategoriesRequest) (*posv1.ReorderCategoriesResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	orderedIDs := make([]uuid.UUID, 0, len(req.Ids))
	for _, s := range req.Ids {
		if id, err := uuid.Parse(s); err == nil {
			orderedIDs = append(orderedIDs, id)
		}
	}
	if err := h.reorderCategoryH.Handle(ctx, tenantID, orderedIDs); err != nil {
		return nil, status.Error(codes.Internal, "failed to reorder categories")
	}
	return &posv1.ReorderCategoriesResponse{}, nil
}

func (h *ProductHandler) SetBranchPrice(ctx context.Context, req *posv1.SetBranchPriceRequest) (*posv1.SetBranchPriceResponse, error) {
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	var overridePrice *int64
	if req.OverridePrice > 0 {
		v := req.OverridePrice
		overridePrice = &v
	}
	if err := h.setBranchPriceH.Handle(ctx, command.SetBranchPriceInput{
		BranchID:       branchID,
		ProductID:      productID,
		OverridePrice:  overridePrice,
		IsAvailable:    req.IsAvailable,
		IdempotencyKey: req.IdempotencyKey,
	}); err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.SetBranchPriceResponse{}, nil
}

func domainProductToProto(p *product.Product) *posv1.Product {
	proto := &posv1.Product{
		Id:          p.ID.String(),
		TenantId:    p.TenantID.String(),
		Sku:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		BasePrice:   p.BasePrice,
		TaxType:     domainTaxTypeToProto(p.TaxType),
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.CategoryID != nil {
		proto.CategoryId = p.CategoryID.String()
	}
	return proto
}

func domainCategoryToProto(c *product.Category) *posv1.Category {
	proto := &posv1.Category{
		Id:        c.ID.String(),
		TenantId:  c.TenantID.String(),
		Name:      c.Name,
		SortOrder: int32(c.SortOrder),
		IsActive:  c.IsActive,
	}
	if c.ParentID != nil {
		proto.ParentId = c.ParentID.String()
	}
	return proto
}

func protoTaxTypeToDomain(t posv1.TaxType) product.TaxType {
	switch t {
	case posv1.TaxType_TAX_TYPE_PPN:
		return product.TaxTypePPN
	case posv1.TaxType_TAX_TYPE_PB1:
		return product.TaxTypePB1
	default:
		return product.TaxTypeNone
	}
}

func domainTaxTypeToProto(t product.TaxType) posv1.TaxType {
	switch t {
	case product.TaxTypePPN:
		return posv1.TaxType_TAX_TYPE_PPN
	case product.TaxTypePB1:
		return posv1.TaxType_TAX_TYPE_PB1
	default:
		return posv1.TaxType_TAX_TYPE_NONE
	}
}

func mapProductError(err error) error {
	switch {
	case errors.Is(err, product.ErrProductNotFound):
		return status.Error(codes.NotFound, "product not found")
	case errors.Is(err, product.ErrSKUAlreadyExists):
		return status.Error(codes.AlreadyExists, "SKU already exists")
	case errors.Is(err, product.ErrProductHasActiveOrders):
		return status.Error(codes.FailedPrecondition, "product has active orders")
	case errors.Is(err, product.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, "base price must be >= 0")
	case errors.Is(err, product.ErrInvalidTaxType):
		return status.Error(codes.InvalidArgument, "invalid tax type")
	case errors.Is(err, product.ErrCategoryNotFound):
		return status.Error(codes.NotFound, "category not found")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
