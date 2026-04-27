package db

import "errors"

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrForbidden           = errors.New("forbidden")
	ErrEmailTaken          = errors.New("email already taken")
	ErrInvalidID           = errors.New("invalid id")
	ErrInvalidArgument     = errors.New("invalid argument")
	ErrNotConsecutiveSlots = errors.New("slots must be consecutive")
)
