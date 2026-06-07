package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedErrors "github.com/xyn-pos/shared/pkg/errors"
)

var (
	errNotFound = stderrors.New("not found")
	errInvalid  = stderrors.New("invalid")
)

var testMappings = []sharedErrors.Mapping{
	{Err: errNotFound, Code: codes.NotFound, Message: "resource not found"},
	{Err: errInvalid, Code: codes.InvalidArgument, Message: "invalid argument"},
}

func TestMapToGRPCStatus_NilError(t *testing.T) {
	assert.NoError(t, sharedErrors.MapToGRPCStatus(nil, testMappings))
}

func TestMapToGRPCStatus_MappedError(t *testing.T) {
	err := sharedErrors.MapToGRPCStatus(errNotFound, testMappings)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "resource not found", st.Message())
}

func TestMapToGRPCStatus_WrappedError(t *testing.T) {
	wrapped := stderrors.Join(stderrors.New("outer"), errInvalid)
	err := sharedErrors.MapToGRPCStatus(wrapped, testMappings)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestMapToGRPCStatus_UnmappedError_ReturnsInternal(t *testing.T) {
	err := sharedErrors.MapToGRPCStatus(stderrors.New("unknown"), testMappings)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestNotFound(t *testing.T) {
	err := sharedErrors.NotFound("thing not found")
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestInvalidArgument(t *testing.T) {
	err := sharedErrors.InvalidArgument("bad input")
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}
