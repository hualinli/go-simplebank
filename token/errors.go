package token

import "errors"

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("token has expired")
	ErrMalformedToken = errors.New("malformed token")
	ErrInvalidKeySize = errors.New("invalid key size: must be at least 32 characters")
)
