package domain

import "errors"

var (
	ErrURLNotFound       = errors.New("url not found")
	ErrInvalidURL        = errors.New("invalid url")
	ErrShortCodeConflict = errors.New("short code generation failed after max retries")
)
