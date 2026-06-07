package order

import (
	"context"

	"github.com/google/uuid"
)

type OrderFilter struct {
	ShiftID   *uuid.UUID
	CashierID *uuid.UUID
	BranchID  *uuid.UUID
	Status    *OrderStatus
	Limit     int
	Offset    int
}

type OrderRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	FindByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*Order, error)
	FindByFilter(ctx context.Context, tenantID uuid.UUID, filter OrderFilter) ([]*Order, int, error)
	Save(ctx context.Context, o *Order) error
	Update(ctx context.Context, o *Order) error
}

type ShiftRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Shift, error)
	FindOpenByBranchAndCashier(ctx context.Context, branchID, cashierID uuid.UUID) (*Shift, error)
	Save(ctx context.Context, s *Shift) error
	Update(ctx context.Context, s *Shift) error
}
