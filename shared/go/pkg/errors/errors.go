package errors

import (
	"errors"
	"fmt"

	"github.com/samber/oops"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sentinel base errors — services extend these with domain-specific errors.
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInternal         = errors.New("internal error")
)

// Wrap wraps an error with oops for rich context. Use at layer boundaries.
func Wrap(err error, msg string) error {
	return oops.Wrapf(err, "%s", msg)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...any) error {
	return oops.Wrapf(err, format, args...)
}

// New creates a new error (convenience wrapper).
func New(msg string) error {
	return fmt.Errorf("%s", msg)
}

// MapToGRPCStatus maps a domain error to a gRPC status.
// Uses errors.Is to walk the error chain.
func MapToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return status.New(codes.NotFound, publicMessage(err))
	case errors.Is(err, ErrAlreadyExists):
		return status.New(codes.AlreadyExists, publicMessage(err))
	case errors.Is(err, ErrInvalidArgument):
		return status.New(codes.InvalidArgument, publicMessage(err))
	case errors.Is(err, ErrUnauthenticated):
		return status.New(codes.Unauthenticated, publicMessage(err))
	case errors.Is(err, ErrPermissionDenied):
		return status.New(codes.PermissionDenied, publicMessage(err))
	default:
		return status.New(codes.Internal, "internal error")
	}
}

// publicMessage extracts the oops public message if present, else uses err.Error().
func publicMessage(err error) string {
	var oopsErr oops.OopsError
	if errors.As(err, &oopsErr) {
		if msg := oopsErr.Public(); msg != "" {
			return msg
		}
	}
	return err.Error()
}
