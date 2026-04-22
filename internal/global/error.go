package global

import "errors"

var (
	ErrLocked error = errors.New("locked")
)
