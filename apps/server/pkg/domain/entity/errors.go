package entity

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrBadRequest     = errors.New("bad request")
	ErrConflict       = errors.New("conflict")
	ErrInternalServer = errors.New("internal server error")

	// ErrUnsupported is returned by a store backend that does not implement a
	// given capability. Used by the ClickHouse store for relational features
	// like datasets/evals/prompts — those require a SQLite or Postgres primary.
	ErrUnsupported = errors.New("operation not supported by this store")
)
