package gapi

import (
	"net/mail"
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func validateCreateUserRequest(reqUsername, reqPassword, reqFullName, reqEmail string) error {
	if reqUsername == "" || !usernameRegex.MatchString(reqUsername) {
		return ErrInvalidRequest
	}
	if len(reqPassword) < 6 {
		return ErrInvalidRequest
	}
	if strings.TrimSpace(reqFullName) == "" {
		return ErrInvalidRequest
	}
	if _, err := mail.ParseAddress(reqEmail); err != nil {
		return ErrInvalidRequest
	}
	return nil
}

func validateLoginUserRequest(reqUsername, reqPassword string) error {
	if reqUsername == "" || !usernameRegex.MatchString(reqUsername) {
		return ErrInvalidRequest
	}
	if len(reqPassword) < 6 {
		return ErrInvalidRequest
	}
	return nil
}
