package valueobject

import "errors"

var (
	ErrInvalidURL  = errors.New("invalid url format")
	ErrInvalidCode = errors.New("invalid short code format")
)
