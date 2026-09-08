package learn

import "errors"

var (
	ErrNotFound      = errors.New("not_found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidStatus = errors.New("invalid_status")
	ErrInvalidInput  = errors.New("invalid_input")
)
