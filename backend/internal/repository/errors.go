// Package repository holds repository-wide errors and helpers mapping PostgreSQL
// (pgx) errors to predictable service-layer values.
package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDBCheckNilConn = errors.New("db connection nil")
var ErrTableDoesNotExist = errors.New("table does not exist")
var ErrUpdateFailed = errors.New("update failed")
var ErrInsertFailed = errors.New("insert failed")
var ErrGetFailed = errors.New("get failed")
var ErrFindFailed = errors.New("find failed")
var ErrNotFound = errors.New("not found")
var ErrNoRecords = errors.New("no records found")
var ErrDuplicateKey = errors.New("duplicate key")
var ErrForeignKeyViolated = errors.New("foreign key violated")
var ErrConflict = errors.New("conflict")
var ErrNotNullViolation = errors.New("not null violated")
var ErrCheckViolation = errors.New("check constraint violated")

// DBErrToErr maps a *pgconn.PgError to a repository sentinel. ok=false when err
// isn't a PostgreSQL error or the code is unknown, in which case the caller
// returns the original err as-is.
//
// pgx.ErrNoRows is not covered here; it's checked separately via errors.Is.
func DBErrToErr(err error) (res error, ok bool) {
	var dbErr *pgconn.PgError
	if !errors.As(err, &dbErr) {
		return nil, false
	}

	detail := dbErr.Detail
	if detail == "" {
		detail = "-"
	}

	switch dbErr.Code {
	case "42P01":
		return fmt.Errorf("%w: %s: %s", ErrTableDoesNotExist, dbErr.Message, detail), true
	case "23503":
		return fmt.Errorf("%w: %s: %s", ErrForeignKeyViolated, dbErr.Message, detail), true
	case "23505":
		return fmt.Errorf("%w: %s: %s", ErrDuplicateKey, dbErr.Message, detail), true
	case "23502":
		return fmt.Errorf("%w: %s: %s", ErrNotNullViolation, dbErr.Message, detail), true
	case "23514":
		return fmt.Errorf("%w: %s: %s", ErrCheckViolation, dbErr.Message, detail), true
	case "23P01", "40001", "40P01":
		return fmt.Errorf("%w: %s: %s", ErrConflict, dbErr.Message, detail), true
	default:
		return nil, false
	}
}

// MapDBErr returns the mapped sentinel for a known PgError, otherwise the original err.
func MapDBErr(err error) error {
	if err == nil {
		return nil
	}
	if mappedErr, ok := DBErrToErr(err); ok {
		return mappedErr
	}
	return err
}
