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
	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// OrderHandler implements posv1.OrderServiceServer.
type OrderHandler struct {
	posv1.UnimplementedOrderServiceServer
	createOrderH        *command.CreateOrderHandler
	addItemH            *command.AddItemHandler
	removeItemH         *command.RemoveItemHandler
	updateItemQuantityH *command.UpdateItemQuantityHandler
	applyDiscountH      *command.ApplyDiscountHandler
	submitOrderH        *command.SubmitOrderHandler
	cancelOrderH        *command.CancelOrderHandler
	openShiftH          *command.OpenShiftHandler
	closeShiftH         *command.CloseShiftHandler
	getOrderH           *query.GetOrderHandler
	listOrdersH         *query.ListOrdersHandler
	getShiftH           *query.GetShiftHandler
}

// NewOrderHandler assembles the order handler.
func NewOrderHandler(
	createH *command.CreateOrderHandler,
	addH *command.AddItemHandler,
	removeH *command.RemoveItemHandler,
	updateQtyH *command.UpdateItemQuantityHandler,
	discountH *command.ApplyDiscountHandler,
	submitH *command.SubmitOrderHandler,
	cancelH *command.CancelOrderHandler,
	openShiftH *command.OpenShiftHandler,
	closeShiftH *command.CloseShiftHandler,
	getOrderH *query.GetOrderHandler,
	listOrdersH *query.ListOrdersHandler,
	getShiftH *query.GetShiftHandler,
) *OrderHandler {
	return &OrderHandler{
		createOrderH:        createH,
		addItemH:            addH,
		removeItemH:         removeH,
		updateItemQuantityH: updateQtyH,
		applyDiscountH:      discountH,
		submitOrderH:        submitH,
		cancelOrderH:        cancelH,
		openShiftH:          openShiftH,
		closeShiftH:         closeShiftH,
		getOrderH:           getOrderH,
		listOrdersH:         listOrdersH,
		getShiftH:           getShiftH,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *posv1.CreateOrderRequest) (*posv1.CreateOrderResponse, error) {
	tenantID, err := tenantIDFromContext(ctx, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	var shiftID *uuid.UUID
	if req.ShiftId != "" {
		id, parseErr := uuid.Parse(req.ShiftId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid shift_id")
		}
		shiftID = &id
	}
	// CashierID comes from verified JWT claims, not the request body, to prevent spoofing.
	// In dev mode (no auth middleware), fall back to req.CashierId.
	cashierID := uuid.Nil
	if claims, ok := claimsFromContext(ctx); ok {
		cashierID = claims.UserID
	} else if req.CashierId != "" {
		cashierID, _ = uuid.Parse(req.CashierId)
	}
	o, err := h.createOrderH.Handle(ctx, command.CreateOrderInput{
		TenantID:       tenantID,
		BranchID:       branchID,
		ShiftID:        shiftID,
		CashierID:      cashierID,
		OrderType:      protoOrderTypeToDomain(req.OrderType),
		TableNumber:    req.TableNumber,
		Notes:          req.Notes,
		IdempotencyKey: req.IdempotencyKey,
		TaxCfg:         order.TaxConfig{Type: order.TaxTypePPN, Rate: 1100},
	})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.CreateOrderResponse{Order: domainOrderToProto(o)}, nil
}

// assertOrderTenant loads the order and verifies the caller's tenant matches.
// Returns nil in dev mode (no claims in context). Uses codes.NotFound on mismatch
// to avoid leaking existence of orders belonging to other tenants.
func (h *OrderHandler) assertOrderTenant(ctx context.Context, orderID uuid.UUID) error {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		return nil
	}
	o, err := h.getOrderH.Handle(ctx, orderID)
	if err != nil {
		return mapOrderError(err)
	}
	if o.TenantID != claims.TenantID {
		return status.Error(codes.NotFound, "order not found")
	}
	return nil
}

func (h *OrderHandler) AddItem(ctx context.Context, req *posv1.AddItemRequest) (*posv1.AddItemResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	var variantID *uuid.UUID
	if req.VariantId != "" {
		id, _ := uuid.Parse(req.VariantId)
		variantID = &id
	}
	addons := make([]order.OrderItemAddon, 0, len(req.Addons))
	for _, a := range req.Addons {
		aid, _ := uuid.Parse(a.AddonId)
		addons = append(addons, order.OrderItemAddon{AddonID: aid, AddonName: a.AddonName, Price: a.Price})
	}
	o, err := h.addItemH.Handle(ctx, command.AddItemInput{
		OrderID:   orderID,
		ProductID: productID,
		VariantID: variantID,
		Quantity:  int(req.Quantity),
		Notes:     req.Notes,
		Addons:    addons,
	})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.AddItemResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) RemoveItem(ctx context.Context, req *posv1.RemoveItemRequest) (*posv1.RemoveItemResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	itemID, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item_id")
	}
	o, err := h.removeItemH.Handle(ctx, command.RemoveItemInput{OrderID: orderID, ItemID: itemID})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.RemoveItemResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) UpdateItemQuantity(ctx context.Context, req *posv1.UpdateItemQuantityRequest) (*posv1.UpdateItemQuantityResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	itemID, err := uuid.Parse(req.ItemId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid item_id")
	}
	o, err := h.updateItemQuantityH.Handle(ctx, command.UpdateItemQuantityInput{
		OrderID: orderID, ItemID: itemID, Quantity: int(req.Quantity),
	})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.UpdateItemQuantityResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) ApplyDiscount(ctx context.Context, req *posv1.ApplyDiscountRequest) (*posv1.ApplyDiscountResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	o, err := h.applyDiscountH.Handle(ctx, command.ApplyDiscountInput{
		OrderID:      orderID,
		DiscountType: protoDiscountTypeToDomain(req.Type),
		Amount:       req.Amount,
	})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.ApplyDiscountResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) SubmitOrder(ctx context.Context, req *posv1.SubmitOrderRequest) (*posv1.SubmitOrderResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	o, err := h.submitOrderH.Handle(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.SubmitOrderResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *posv1.CancelOrderRequest) (*posv1.CancelOrderResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	if err := h.assertOrderTenant(ctx, orderID); err != nil {
		return nil, err
	}
	if err := h.cancelOrderH.Handle(ctx, orderID, req.Reason); err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.CancelOrderResponse{}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *posv1.GetOrderRequest) (*posv1.GetOrderResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	o, err := h.getOrderH.Handle(ctx, orderID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.GetOrderResponse{Order: domainOrderToProto(o)}, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *posv1.ListOrdersRequest) (*posv1.ListOrdersResponse, error) {
	tenantID, err := tenantIDFromContext(ctx, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	filter := order.OrderFilter{
		Limit: int(req.Pagination.GetLimit()),
	}
	if req.ShiftId != "" {
		id, _ := uuid.Parse(req.ShiftId)
		filter.ShiftID = &id
	}
	if req.Status != posv1.OrderStatus_ORDER_STATUS_UNSPECIFIED {
		s := protoOrderStatusToDomain(req.Status)
		filter.Status = &s
	}
	result, err := h.listOrdersH.Handle(ctx, tenantID, filter)
	if err != nil {
		return nil, mapOrderError(err)
	}
	protoOrders := make([]*posv1.Order, len(result.Orders))
	for i, o := range result.Orders {
		protoOrders[i] = domainOrderToProto(o)
	}
	return &posv1.ListOrdersResponse{Orders: protoOrders}, nil
}

func (h *OrderHandler) OpenShift(ctx context.Context, req *posv1.OpenShiftRequest) (*posv1.OpenShiftResponse, error) {
	tenantID, err := tenantIDFromContext(ctx, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	branchID, err := uuid.Parse(req.BranchId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid branch_id")
	}
	cashierID, _ := uuid.Parse(req.CashierId)
	s, err := h.openShiftH.Handle(ctx, command.OpenShiftInput{
		TenantID:    tenantID,
		BranchID:    branchID,
		CashierID:   cashierID,
		OpeningCash: req.OpeningCash,
	})
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.OpenShiftResponse{Shift: domainShiftToProto(s)}, nil
}

func (h *OrderHandler) CloseShift(ctx context.Context, req *posv1.CloseShiftRequest) (*posv1.CloseShiftResponse, error) {
	shiftID, err := uuid.Parse(req.ShiftId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid shift_id")
	}
	s, err := h.closeShiftH.Handle(ctx, shiftID, req.ClosingCash)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.CloseShiftResponse{Shift: domainShiftToProto(s)}, nil
}

func (h *OrderHandler) GetShift(ctx context.Context, req *posv1.GetShiftRequest) (*posv1.GetShiftResponse, error) {
	shiftID, err := uuid.Parse(req.ShiftId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid shift_id")
	}
	s, err := h.getShiftH.Handle(ctx, shiftID)
	if err != nil {
		return nil, mapOrderError(err)
	}
	return &posv1.GetShiftResponse{Shift: domainShiftToProto(s)}, nil
}

// --- proto ↔ domain conversion helpers ---

func domainOrderToProto(o *order.Order) *posv1.Order {
	proto := &posv1.Order{
		Id:             o.ID.String(),
		TenantId:       o.TenantID.String(),
		BranchId:       o.BranchID.String(),
		CashierId:      o.CashierID.String(),
		OrderNumber:    o.OrderNumber,
		OrderType:      domainOrderTypeToProto(o.OrderType),
		TableNumber:    o.TableNumber,
		Status:         domainOrderStatusToProto(o.Status),
		Subtotal:       o.Subtotal,
		TaxAmount:      o.TaxAmount,
		DiscountAmount: o.DiscountAmount,
		Total:          o.Total,
		Notes:          o.Notes,
		CreatedAt:      o.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      o.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if o.ShiftID != nil {
		proto.ShiftId = o.ShiftID.String()
	}
	if o.Discount != nil {
		proto.DiscountType = domainDiscountTypeToProto(o.Discount.Type)
		proto.DiscountValue = o.Discount.Amount
	}
	items := make([]*posv1.OrderItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = domainOrderItemToProto(item)
	}
	proto.Items = items
	return proto
}

func domainOrderItemToProto(item order.OrderItem) *posv1.OrderItem {
	proto := &posv1.OrderItem{
		Id:          item.ID.String(),
		ProductId:   item.ProductID.String(),
		ProductName: item.ProductName,
		VariantName: item.VariantName,
		UnitPrice:   item.UnitPrice,
		Quantity:    int32(item.Quantity),
		Subtotal:    item.Subtotal,
		Notes:       item.Notes,
	}
	if item.VariantID != nil {
		proto.VariantId = item.VariantID.String()
	}
	addons := make([]*posv1.OrderItemAddon, len(item.Addons))
	for i, a := range item.Addons {
		addons[i] = &posv1.OrderItemAddon{
			AddonId:   a.AddonID.String(),
			AddonName: a.AddonName,
			Price:     a.Price,
		}
	}
	proto.Addons = addons
	return proto
}

func domainShiftToProto(s *order.Shift) *posv1.Shift {
	proto := &posv1.Shift{
		Id:          s.ID.String(),
		TenantId:    s.TenantID.String(),
		BranchId:    s.BranchID.String(),
		CashierId:   s.CashierID.String(),
		Status:      domainShiftStatusToProto(s.Status),
		OpenedAt:    s.OpenedAt.Format("2006-01-02T15:04:05Z"),
		OpeningCash: s.OpeningCash,
	}
	if s.ClosedAt != nil {
		proto.ClosedAt = s.ClosedAt.Format("2006-01-02T15:04:05Z")
	}
	if s.ClosingCash != nil {
		proto.ClosingCash = *s.ClosingCash
	}
	return proto
}

func protoOrderTypeToDomain(t posv1.OrderType) order.OrderType {
	switch t {
	case posv1.OrderType_ORDER_TYPE_TAKEAWAY:
		return order.OrderTypeTakeaway
	case posv1.OrderType_ORDER_TYPE_DELIVERY:
		return order.OrderTypeDelivery
	default:
		return order.OrderTypeDineIn
	}
}

func domainOrderTypeToProto(t order.OrderType) posv1.OrderType {
	switch t {
	case order.OrderTypeTakeaway:
		return posv1.OrderType_ORDER_TYPE_TAKEAWAY
	case order.OrderTypeDelivery:
		return posv1.OrderType_ORDER_TYPE_DELIVERY
	default:
		return posv1.OrderType_ORDER_TYPE_DINE_IN
	}
}

func domainOrderStatusToProto(s order.OrderStatus) posv1.OrderStatus {
	switch s {
	case order.StatusDraft:
		return posv1.OrderStatus_ORDER_STATUS_DRAFT
	case order.StatusPendingPayment:
		return posv1.OrderStatus_ORDER_STATUS_PENDING_PAYMENT
	case order.StatusPaid:
		return posv1.OrderStatus_ORDER_STATUS_PAID
	case order.StatusCancelled:
		return posv1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return posv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func protoOrderStatusToDomain(s posv1.OrderStatus) order.OrderStatus {
	switch s {
	case posv1.OrderStatus_ORDER_STATUS_DRAFT:
		return order.StatusDraft
	case posv1.OrderStatus_ORDER_STATUS_PENDING_PAYMENT:
		return order.StatusPendingPayment
	case posv1.OrderStatus_ORDER_STATUS_PAID:
		return order.StatusPaid
	case posv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return order.StatusCancelled
	default:
		return order.StatusDraft
	}
}

func protoDiscountTypeToDomain(t posv1.DiscountType) order.DiscountType {
	if t == posv1.DiscountType_DISCOUNT_TYPE_PERCENT {
		return order.DiscountTypePercent
	}
	return order.DiscountTypeFixed
}

func domainDiscountTypeToProto(t order.DiscountType) posv1.DiscountType {
	if t == order.DiscountTypePercent {
		return posv1.DiscountType_DISCOUNT_TYPE_PERCENT
	}
	return posv1.DiscountType_DISCOUNT_TYPE_FIXED
}

func domainShiftStatusToProto(s order.ShiftStatus) posv1.ShiftStatus {
	if s == order.ShiftStatusClosed {
		return posv1.ShiftStatus_SHIFT_STATUS_CLOSED
	}
	return posv1.ShiftStatus_SHIFT_STATUS_OPEN
}

// mapOrderError translates domain/application errors to gRPC status codes.
func mapOrderError(err error) error {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		return status.Error(codes.NotFound, "order not found")
	case errors.Is(err, order.ErrOrderNotDraft):
		return status.Error(codes.FailedPrecondition, "order is not in draft state")
	case errors.Is(err, order.ErrOrderEmpty):
		return status.Error(codes.InvalidArgument, "order has no items")
	case errors.Is(err, order.ErrInvalidStatusTransition):
		return status.Error(codes.FailedPrecondition, "invalid order status transition")
	case errors.Is(err, order.ErrItemNotFound):
		return status.Error(codes.NotFound, "order item not found")
	case errors.Is(err, order.ErrShiftNotFound):
		return status.Error(codes.NotFound, "shift not found")
	case errors.Is(err, order.ErrShiftAlreadyClosed):
		return status.Error(codes.FailedPrecondition, "shift is already closed")
	case errors.Is(err, order.ErrShiftAlreadyOpen):
		return status.Error(codes.AlreadyExists, "cashier already has an open shift")
	case errors.Is(err, order.ErrDiscountExceedsSubtotal):
		return status.Error(codes.InvalidArgument, "discount exceeds order subtotal")
	case errors.Is(err, order.ErrDiscountPercentInvalid):
		return status.Error(codes.InvalidArgument, "percent discount must be 0.01%–100%")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
