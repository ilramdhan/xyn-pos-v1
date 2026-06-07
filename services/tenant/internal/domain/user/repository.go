package user

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the port for persisting and retrieving User aggregates.
type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error)
	FindByKeycloakID(ctx context.Context, keycloakID string) (*User, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*User, error)
	Save(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	SavePIN(ctx context.Context, userID uuid.UUID, pinHash string) error
	FindPINHash(ctx context.Context, userID uuid.UUID) (string, error)
}
