package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	sharederrors "github.com/xyn-pos/shared/pkg/errors"
)

func TestMapSentinelToGRPCStatus_NilError(t *testing.T) {
	s := sharederrors.MapSentinelToGRPCStatus(nil)
	assert.Equal(t, codes.OK, s.Code())
}

func TestMapSentinelToGRPCStatus_NotFound(t *testing.T) {
	err := fmt.Errorf("tenant: %w", sharederrors.ErrNotFound)
	s := sharederrors.MapSentinelToGRPCStatus(err)
	assert.Equal(t, codes.NotFound, s.Code())
}

func TestMapSentinelToGRPCStatus_AlreadyExists(t *testing.T) {
	err := fmt.Errorf("slug: %w", sharederrors.ErrAlreadyExists)
	s := sharederrors.MapSentinelToGRPCStatus(err)
	assert.Equal(t, codes.AlreadyExists, s.Code())
}

func TestMapSentinelToGRPCStatus_Unknown(t *testing.T) {
	err := fmt.Errorf("some random error")
	s := sharederrors.MapSentinelToGRPCStatus(err)
	assert.Equal(t, codes.Internal, s.Code())
}

func TestMapSentinelToGRPCStatus_InvalidArgument(t *testing.T) {
	err := fmt.Errorf("name: %w", sharederrors.ErrInvalidArgument)
	s := sharederrors.MapSentinelToGRPCStatus(err)
	assert.Equal(t, codes.InvalidArgument, s.Code())
}
