// Reference: https://github.com/jackc/pgerrcode/blob/master/errcode.go
package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TODO: 在判断逻辑中增加Unit Test的错误，让错误类型的判断逻辑也能被测试覆盖到，而且不依赖pgx
var (
	ErrRecordNotFound      = errors.New("record not found")
	ErrForeignKeyViolation = errors.New("foreign key violation")
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func IsForeignKeyViolationError(err error) bool {
	return false
}

func IsUniqueViolationError(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok { // Go 1.26 New Feature: errors.AsType
		return pgErr.Code == "23505"
	}
	return false
}

func IsInternalError(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "XX000" || pgErr.Code == "XX001" || pgErr.Code == "XX002"
	}
	return false
}
