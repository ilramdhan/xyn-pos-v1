package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	sharederrors "github.com/xyn-pos/shared/pkg/errors"
	"google.golang.org/grpc/codes"
)

func TestMapToGRPCStatus_NilError(t *testing.T) {
	s := sharederrors.MapToGRPCStatus(nil)
	assert.Equal(t, codes.OK, s.Code())
}

func TestMapToGRPCStatus_NotFound(t *testing.T) {
	err := fmt.Errorf("tenant: %w", sharederrors.ErrNotFound)
	s := sharederrors.MapToGRPCStatus(err)
	assert.Equal(t, codes.NotFound, s.Code())
}

func TestMapToGRPCStatus_AlreadyExists(t *testing.T) {
	err := fmt.Errorf("slug: %w", sharederrors.ErrAlreadyExists)
	s := sharederrors.MapToGRPCStatus(err)
	assert.Equal(t, codes.AlreadyExists, s.Code())
}

func TestMapToGRPCStatus_Unknown(t *testing.T) {
	err := fmt.Errorf("some random error")
	s := sharederrors.MapToGRPCStatus(err)
	assert.Equal(t, codes.Internal, s.Code())
}

func TestMapToGRPCStatus_InvalidArgument(t *testing.T) {
	err := fmt.Errorf("name: %w", sharederrors.ErrInvalidArgument)
	s := sharederrors.MapToGRPCStatus(err)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}
