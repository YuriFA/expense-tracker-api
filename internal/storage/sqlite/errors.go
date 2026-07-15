package sqlite

import (
	"errors"

	"github.com/mattn/go-sqlite3"
)

func isFKViolationError(err error) bool {
	if sqliteErr, ok := errors.AsType[sqlite3.Error](err); ok {
		return sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey
	}

	return false
}

func isUniqueConstraintViolation(err error) bool {
	if sqliteErr, ok := errors.AsType[sqlite3.Error](err); ok {
		return sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
	}

	return false
}
