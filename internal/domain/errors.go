package domain

import "errors"

var (
	ErrNotFound = errors.New("resource not found")

	ErrInvalidInput = errors.New("invalid input")

	ErrConflict = errors.New("conflict: resource already exists or violates constraints")

	ErrCyclicTree = errors.New("conflict: cannot move department into its own subtree")
)
