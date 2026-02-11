package grpcutil

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrInvalidArgument returns invalid argument error
func ErrInvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

// ErrNotFound returns not found error
func ErrNotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

// ErrUnauthenticated returns unauthenticated error
func ErrUnauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

// ErrPermissionDenied returns permission denied error
func ErrPermissionDenied(msg string) error {
	return status.Error(codes.PermissionDenied, msg)
}

// ErrInternal returns internal error
func ErrInternal(msg string) error {
	return status.Error(codes.Internal, msg)
}

// ErrAlreadyExists returns already exists error
func ErrAlreadyExists(msg string) error {
	return status.Error(codes.AlreadyExists, msg)
}
