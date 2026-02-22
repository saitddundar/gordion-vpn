package grpcutil

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrInvalidArgument(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func ErrNotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

func ErrUnauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

func ErrPermissionDenied(msg string) error {
	return status.Error(codes.PermissionDenied, msg)
}

func ErrInternal(msg string) error {
	return status.Error(codes.Internal, msg)
}

func ErrAlreadyExists(msg string) error {
	return status.Error(codes.AlreadyExists, msg)
}
