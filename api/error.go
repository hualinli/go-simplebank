package api

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func errResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

// 公共错误
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInternalError  = errors.New("internal server error")
	ErrUnknownError   = errors.New("unknown error")
	ErrUnauthorized   = errors.New("unauthorized")
)

// User相关错误
var (
	ErrUserOrEmailAlreadyExists = errors.New("username or email already exists")
	ErrUserNotFound             = errors.New("user not found")
	ErrInvalidPassword          = errors.New("invalid password")
	ErrUserNotMatch             = errors.New("user doesn't match the authenticated user")
	ErrPasswordMustBeDifferent  = errors.New("new password must be different from the old password")
)

// Account相关错误
var (
	ErrAccountNotFound        = errors.New("account not found")
	ErrAccountNotMatch        = errors.New("account doesn't belong to the authenticated user")
	ErrAccountAlreadyExists   = errors.New("account with the same currency already exists")
	ErrAccountForbidden       = errors.New("account doesn't belong to the authenticated user")
	ErrAccountCannotBeDeleted = errors.New("account cannot be deleted because it has non-zero balance or has associated entries")
)
