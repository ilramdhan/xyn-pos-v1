package order

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusDraft          OrderStatus = "draft"
	StatusPendingPayment OrderStatus = "pending_payment"
	StatusPaid           OrderStatus = "paid"
	StatusCancelled      OrderStatus = "cancelled"
	StatusParked         OrderStatus = "parked"
)

type OrderType string

const (
	OrderTypeDineIn   OrderType = "dine_in"
	OrderTypeTakeaway OrderType = "takeaway"
	OrderTypeDelivery OrderType = "delivery"
)

type DiscountType string

const (
	DiscountTypeFixed   DiscountType = "fixed"
	DiscountTypePercent DiscountType = "percent"
)

type Discount struct {
	Type   DiscountType
	Amount int64
}

type OrderItemAddon struct {
	AddonID   uuid.UUID
	AddonName string
	Price     int64
}

type OrderItem struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	ProductID   uuid.UUID
	VariantID   *uuid.UUID
	ProductName string
	VariantName string
	UnitPrice   int64
	Quantity    int
	Subtotal    int64
	Notes       string
	Addons      []OrderItemAddon
}

type Order struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	BranchID       uuid.UUID
	ShiftID        *uuid.UUID
	CashierID      uuid.UUID
	OrderNumber    string
	OrderType      OrderType
	TableNumber    string
	Status         OrderStatus
	Items          []OrderItem
	Discount       *Discount
	Subtotal       int64
	TaxAmount      int64
	DiscountAmount int64
	Total          int64
	TaxCfg         TaxConfig
	IdempotencyKey string
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	events         []DomainEvent
}

func NewOrder(tenantID, branchID uuid.UUID, shiftID *uuid.UUID,
	cashierID uuid.UUID, orderNumber string, orderType OrderType,
	tableNumber, idempotencyKey string, taxCfg TaxConfig) (*Order, error) {

	now := time.Now().UTC()
	return &Order{
		ID:             uuid.New(),
		TenantID:       tenantID,
		BranchID:       branchID,
		ShiftID:        shiftID,
		CashierID:      cashierID,
		OrderNumber:    orderNumber,
		OrderType:      orderType,
		TableNumber:    tableNumber,
		Status:         StatusDraft,
		TaxCfg:         taxCfg,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (o *Order) AddItem(item OrderItem) error {
	if o.Status != StatusDraft {
		return ErrOrderNotDraft
	}
	item.ID = uuid.New()
	item.OrderID = o.ID
	item.Subtotal = item.UnitPrice * int64(item.Quantity)
	o.Items = append(o.Items, item)
	o.recalculate()
	return nil
}

func (o *Order) RemoveItem(itemID uuid.UUID) error {
	if o.Status != StatusDraft {
		return ErrOrderNotDraft
	}
	for i, item := range o.Items {
		if item.ID == itemID {
			o.Items = append(o.Items[:i], o.Items[i+1:]...)
			o.recalculate()
			return nil
		}
	}
	return ErrItemNotFound
}

func (o *Order) UpdateItemQuantity(itemID uuid.UUID, qty int) error {
	if o.Status != StatusDraft {
		return ErrOrderNotDraft
	}
	for i, item := range o.Items {
		if item.ID == itemID {
			o.Items[i].Quantity = qty
			o.Items[i].Subtotal = item.UnitPrice * int64(qty)
			o.recalculate()
			return nil
		}
	}
	return ErrItemNotFound
}

func (o *Order) ApplyDiscount(d Discount) error {
	if o.Status != StatusDraft {
		return ErrOrderNotDraft
	}
	if d.Type == DiscountTypeFixed && d.Amount > o.Subtotal {
		return ErrDiscountExceedsSubtotal
	}
	if d.Type == DiscountTypePercent && (d.Amount <= 0 || d.Amount > 10000) {
		return ErrDiscountPercentInvalid
	}
	o.Discount = &d
	o.recalculate()
	return nil
}

func (o *Order) Submit() error {
	if o.Status != StatusDraft {
		return fmt.Errorf("Submit: %w", ErrInvalidStatusTransition)
	}
	if len(o.Items) == 0 {
		return ErrOrderEmpty
	}
	o.Status = StatusPendingPayment
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (o *Order) MarkPaid() error {
	if o.Status != StatusPendingPayment {
		return fmt.Errorf("MarkPaid: %w", ErrInvalidStatusTransition)
	}
	o.Status = StatusPaid
	o.UpdatedAt = time.Now().UTC()

	items := make([]OrderPaidItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderPaidItem{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		}
	}
	o.events = append(o.events, OrderPaidEvent{
		OrderID:    o.ID,
		TenantID:   o.TenantID,
		BranchID:   o.BranchID,
		CashierID:  o.CashierID,
		Items:      items,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

func (o *Order) Cancel(reason string) error {
	if o.Status != StatusDraft && o.Status != StatusPendingPayment {
		return fmt.Errorf("Cancel: %w", ErrInvalidStatusTransition)
	}
	o.Status = StatusCancelled
	o.Notes = reason
	o.UpdatedAt = time.Now().UTC()
	o.events = append(o.events, OrderCancelledEvent{
		OrderID:    o.ID,
		TenantID:   o.TenantID,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

// Park sets the order to parked status, suspending it for later resumption.
// Only orders in DRAFT status can be parked.
func (o *Order) Park() error {
	if o.Status != StatusDraft {
		return fmt.Errorf("Park: %w", ErrInvalidStatusTransition)
	}
	o.Status = StatusParked
	o.UpdatedAt = time.Now().UTC()
	o.events = append(o.events, OrderParkedEvent{
		OrderID:    o.ID,
		TenantID:   o.TenantID,
		BranchID:   o.BranchID,
		CashierID:  o.CashierID,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

// Resume restores a parked order to DRAFT status so items can be modified.
func (o *Order) Resume() error {
	if o.Status != StatusParked {
		return fmt.Errorf("Resume: %w", ErrOrderNotParked)
	}
	o.Status = StatusDraft
	o.UpdatedAt = time.Now().UTC()
	o.events = append(o.events, OrderResumedEvent{
		OrderID:    o.ID,
		TenantID:   o.TenantID,
		BranchID:   o.BranchID,
		CashierID:  o.CashierID,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

func (o *Order) PopEvents() []DomainEvent {
	evs := o.events
	o.events = nil
	return evs
}

func (o *Order) recalculate() {
	var subtotal int64
	for _, item := range o.Items {
		subtotal += item.Subtotal
		for _, addon := range item.Addons {
			subtotal += addon.Price * int64(item.Quantity)
		}
	}
	o.Subtotal = subtotal

	taxAmount, total := o.TaxCfg.Calculate(subtotal)
	o.TaxAmount = taxAmount

	var discountAmount int64
	if o.Discount != nil {
		switch o.Discount.Type {
		case DiscountTypeFixed:
			discountAmount = o.Discount.Amount
		case DiscountTypePercent:
			discountAmount = subtotal * o.Discount.Amount / 10000
		}
	}
	o.DiscountAmount = discountAmount
	o.Total = total - discountAmount
	o.UpdatedAt = time.Now().UTC()
}
