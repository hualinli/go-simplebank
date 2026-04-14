package gapi

import (
	"errors"

	"github.com/hualinli/go-simplebank/token"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInternal       = errors.New("internal error")
	ErrInvalidRequest = errors.New("invalid request")
)

var (
	ErrUserExists      = errors.New("username or email already exists")
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnauthorized    = errors.New("unauthorized")
)

func toRPCError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrUserExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrInvalidPassword):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case isTokenError(err):
		return status.Error(codes.Unauthenticated, token.ErrInvalidToken.Error())
	default:
		return status.Error(codes.Internal, ErrInternal.Error())
	}
}
