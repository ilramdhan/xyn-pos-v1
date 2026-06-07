package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/xyn-pos/gen/go/inventory/v1"
	"github.com/xyn-pos/services/inventory/internal/application/command"
	"github.com/xyn-pos/services/inventory/internal/application/query"
	"github.com/xyn-pos/services/inventory/internal/domain/stock"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// InventoryHandler implements InventoryServiceServer.
type InventoryHandler struct {
	inventoryv1.UnimplementedInventoryServiceServer
	adjustH    *command.AdjustStockHandler
	setBOMH    *command.SetBOMHandler
	getStockH  *query.GetStockHandler
	listStockH *query.ListStockHandler
	getBOMH    *query.GetBOMHandler
}

// NewInventoryHandler constructs an InventoryHandler with all application handlers.
func NewInventoryHandler(
	adjustH *command.AdjustStockHandler,
	setBOMH *command.SetBOMHandler,
	getStockH *query.GetStockHandler,
	listStockH *query.ListStockHandler,
	getBOMH *query.GetBOMHandler,
) *InventoryHandler {
	return &InventoryHandler{
		adjustH:    adjustH,
		setBOMH:    setBOMH,
		getStockH:  getStockH,
		listStockH: listStockH,
		getBOMH:    getBOMH,
	}
}

func claimsFromContext(ctx context.Context) (*sharedauth.Claims, bool) {
	return middleware.ClaimsFromContext(ctx)
}

func tenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		return uuid.Nil, errors.New("missing auth context")
	}
	return claims.TenantID, nil
}

// GetStock retrieves the current stock level for a product/variant at a branch.
func (h *InventoryHandler) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	var variantID *uuid.UUID
	if req.VariantId != "" {
		vid, err := uuid.Parse(req.VariantId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid variant_id")
		}
		variantID = &vid
	}
	s, err := h.getStockH.Handle(ctx, tenantID, branchID, productID, variantID)
	if err != nil {
		return nil, mapStockError(err)
	}
	return &inventoryv1.GetStockResponse{Stock: domainLedgerToProto(s)}, nil
}

// ListStock returns all stock ledgers for a branch, optionally filtered to low-stock items.
func (h *InventoryHandler) ListStock(ctx context.Context, req *inventoryv1.ListStockRequest) (*inventoryv1.ListStockResponse, error) {
	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	stocks, err := h.listStockH.Handle(ctx, tenantID, branchID, stock.StockFilter{LowStockOnly: req.LowStockOnly})
	if err != nil {
		return nil, mapStockError(err)
	}
	protos := make([]*inventoryv1.StockLedger, len(stocks))
	for i, s := range stocks {
		protos[i] = domainLedgerToProto(s)
	}
	return &inventoryv1.ListStockResponse{Stocks: protos}, nil
}

// AdjustStock applies a signed delta to a stock ledger; auto-creates the ledger if absent.
func (h *InventoryHandler) AdjustStock(ctx context.Context, req *inventoryv1.AdjustStockRequest) (*inventoryv1.AdjustStockResponse, error) {
	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	var variantID *uuid.UUID
	if req.VariantId != "" {
		vid, err := uuid.Parse(req.VariantId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid variant_id")
		}
		variantID = &vid
	}
	s, err := h.adjustH.Handle(ctx, command.AdjustStockInput{
		TenantID:  tenantID,
		BranchID:  branchID,
		ProductID: productID,
		VariantID: variantID,
		Delta:     req.Delta,
		Unit:      "pcs", // proto AdjustStockRequest has no unit field; default to "pcs"
		Note:      req.Note,
	})
	if err != nil {
		return nil, mapStockError(err)
	}
	return &inventoryv1.AdjustStockResponse{Stock: domainLedgerToProto(s)}, nil
}

// GetBOM retrieves Bill-of-Materials recipes for a product.
func (h *InventoryHandler) GetBOM(ctx context.Context, req *inventoryv1.GetBOMRequest) (*inventoryv1.GetBOMResponse, error) {
	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	recipes, err := h.getBOMH.Handle(ctx, tenantID, productID)
	if err != nil {
		return nil, mapStockError(err)
	}
	protos := make([]*inventoryv1.BOMRecipe, len(recipes))
	for i, r := range recipes {
		protos[i] = domainBOMToProto(r)
	}
	return &inventoryv1.GetBOMResponse{Recipes: protos}, nil
}

// SetBOM atomically replaces the BOM for a product.
func (h *InventoryHandler) SetBOM(ctx context.Context, req *inventoryv1.SetBOMRequest) (*inventoryv1.SetBOMResponse, error) {
	tenantID, err := tenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	lines := make([]command.BOMLine, 0, len(req.Recipes))
	for _, r := range req.Recipes {
		ingID, err := uuid.Parse(r.IngredientProductId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid ingredient_product_id")
		}
		lines = append(lines, command.BOMLine{
			IngredientProductID: ingID,
			QuantityPerUnit:     r.QuantityPerUnit,
			Unit:                r.Unit,
		})
	}
	if err := h.setBOMH.Handle(ctx, command.SetBOMInput{
		TenantID:  tenantID,
		ProductID: productID,
		Lines:     lines,
	}); err != nil {
		return nil, mapStockError(err)
	}
	return &inventoryv1.SetBOMResponse{}, nil
}

// domainLedgerToProto converts a domain StockLedger to its proto representation.
func domainLedgerToProto(s *stock.StockLedger) *inventoryv1.StockLedger {
	proto := &inventoryv1.StockLedger{
		Id:                s.ID.String(),
		TenantId:          s.TenantID.String(),
		BranchId:          s.BranchID.String(),
		ProductId:         s.ProductID.String(),
		Quantity:          s.Quantity,
		Unit:              s.Unit,
		LowStockThreshold: s.LowStockThreshold,
		UpdatedAt:         s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if s.VariantID != nil {
		proto.VariantId = s.VariantID.String()
	}
	return proto
}

// domainBOMToProto converts a domain BOMRecipe to its proto representation.
func domainBOMToProto(r *stock.BOMRecipe) *inventoryv1.BOMRecipe {
	return &inventoryv1.BOMRecipe{
		Id:                  r.ID.String(),
		TenantId:            r.TenantID.String(),
		ProductId:           r.ProductID.String(),
		IngredientProductId: r.IngredientProductID.String(),
		QuantityPerUnit:     r.QuantityPerUnit,
		Unit:                r.Unit,
	}
}

// mapStockError translates domain errors to gRPC status codes.
func mapStockError(err error) error {
	switch {
	case errors.Is(err, stock.ErrStockLedgerNotFound):
		return status.Error(codes.NotFound, "stock ledger not found")
	case errors.Is(err, stock.ErrBOMRecipeNotFound):
		return status.Error(codes.NotFound, "BOM recipe not found")
	case errors.Is(err, stock.ErrInsufficientStock):
		return status.Error(codes.FailedPrecondition, "insufficient stock")
	case errors.Is(err, stock.ErrInvalidDelta), errors.Is(err, stock.ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, "invalid quantity")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
