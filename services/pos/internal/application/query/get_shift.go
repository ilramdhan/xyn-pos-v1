package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// GetShiftHandler retrieves a cashier shift by its ID.
type GetShiftHandler struct{ repo order.ShiftRepository }

// NewGetShiftHandler constructs a GetShiftHandler.
func NewGetShiftHandler(repo order.ShiftRepository) *GetShiftHandler {
	return &GetShiftHandler{repo: repo}
}

// Handle executes the get-shift query.
func (h *GetShiftHandler) Handle(ctx context.Context, id uuid.UUID) (*order.Shift, error) {
	s, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetShift: %w", err)
	}
	return s, nil
}
