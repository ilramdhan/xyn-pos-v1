package order

import (
	"time"

	"github.com/google/uuid"
)

type ShiftStatus string

const (
	ShiftStatusOpen   ShiftStatus = "open"
	ShiftStatusClosed ShiftStatus = "closed"
)

type Shift struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	BranchID    uuid.UUID
	CashierID   uuid.UUID
	Status      ShiftStatus
	OpenedAt    time.Time
	ClosedAt    *time.Time
	OpeningCash int64
	ClosingCash *int64
}

func NewShift(tenantID, branchID, cashierID uuid.UUID, openingCash int64) *Shift {
	return &Shift{
		ID:          uuid.New(),
		TenantID:    tenantID,
		BranchID:    branchID,
		CashierID:   cashierID,
		Status:      ShiftStatusOpen,
		OpenedAt:    time.Now().UTC(),
		OpeningCash: openingCash,
	}
}

func (s *Shift) Close(closingCash int64) error {
	if s.Status == ShiftStatusClosed {
		return ErrShiftAlreadyClosed
	}
	now := time.Now().UTC()
	s.Status = ShiftStatusClosed
	s.ClosedAt = &now
	s.ClosingCash = &closingCash
	return nil
}
