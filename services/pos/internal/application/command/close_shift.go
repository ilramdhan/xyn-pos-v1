package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// CloseShiftHandler closes an open cashier shift and records the closing cash amount.
type CloseShiftHandler struct{ repo order.ShiftRepository }

// NewCloseShiftHandler constructs a CloseShiftHandler.
func NewCloseShiftHandler(repo order.ShiftRepository) *CloseShiftHandler {
	return &CloseShiftHandler{repo: repo}
}

// Handle executes the close-shift use case.
func (h *CloseShiftHandler) Handle(ctx context.Context, shiftID uuid.UUID, closingCash int64) (*order.Shift, error) {
	s, err := h.repo.FindByID(ctx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("CloseShift find: %w", err)
	}
	if err := s.Close(closingCash); err != nil {
		return nil, fmt.Errorf("CloseShift domain: %w", err)
	}
	if err := h.repo.Update(ctx, s); err != nil {
		return nil, fmt.Errorf("CloseShift update: %w", err)
	}
	return s, nil
}
