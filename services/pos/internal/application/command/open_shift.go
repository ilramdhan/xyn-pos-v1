package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// OpenShiftInput holds all parameters to open a cashier shift.
type OpenShiftInput struct {
	TenantID    uuid.UUID
	BranchID    uuid.UUID
	CashierID   uuid.UUID
	OpeningCash int64
}

// OpenShiftHandler opens a new cashier shift, enforcing the one-open-shift-per-cashier rule.
type OpenShiftHandler struct{ repo order.ShiftRepository }

// NewOpenShiftHandler constructs an OpenShiftHandler.
func NewOpenShiftHandler(repo order.ShiftRepository) *OpenShiftHandler {
	return &OpenShiftHandler{repo: repo}
}

// Handle executes the open-shift use case.
func (h *OpenShiftHandler) Handle(ctx context.Context, in OpenShiftInput) (*order.Shift, error) {
	// Reject if the cashier already has an open shift at this branch.
	_, err := h.repo.FindOpenByBranchAndCashier(ctx, in.BranchID, in.CashierID)
	if err == nil {
		return nil, fmt.Errorf("OpenShift: %w", order.ErrShiftAlreadyOpen)
	}
	if !errors.Is(err, order.ErrShiftNotFound) {
		return nil, fmt.Errorf("OpenShift check existing: %w", err)
	}

	s := order.NewShift(in.TenantID, in.BranchID, in.CashierID, in.OpeningCash)
	if err := h.repo.Save(ctx, s); err != nil {
		return nil, fmt.Errorf("OpenShift save: %w", err)
	}
	return s, nil
}
