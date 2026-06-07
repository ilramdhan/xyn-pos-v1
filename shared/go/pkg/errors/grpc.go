package errors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Mapping describes a sentinel error → gRPC code + message pairing.
type Mapping struct {
	Err     error
	Code    codes.Code
	Message string
}

// MapToGRPCStatus maps err to a gRPC status using the provided mappings.
// Falls back to codes.Internal if no mapping matches.
// Returns nil if err is nil.
func MapToGRPCStatus(err error, mappings []Mapping) error {
	if err == nil {
		return nil
	}
	for _, m := range mappings {
		if errors.Is(err, m.Err) {
			return status.Error(m.Code, m.Message)
		}
	}
	return status.Error(codes.Internal, "internal error")
}

// NotFound returns a standard gRPC NotFound status.
func NotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

// InvalidArgument returns a standard gRPC InvalidArgument status.
func InvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

// Unauthenticated returns a standard gRPC Unauthenticated status.
func Unauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

// PermissionDenied returns a standard gRPC PermissionDenied status.
func PermissionDenied(msg string) error {
	return status.Error(codes.PermissionDenied, msg)
}

// AlreadyExists returns a standard gRPC AlreadyExists status.
func AlreadyExists(msg string) error {
	return status.Error(codes.AlreadyExists, msg)
}

// FailedPrecondition returns a standard gRPC FailedPrecondition status.
func FailedPrecondition(msg string) error {
	return status.Error(codes.FailedPrecondition, msg)
}

// Internal returns a standard gRPC Internal status.
func Internal(msg string) error {
	return status.Error(codes.Internal, msg)
}
