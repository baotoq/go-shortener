package domain

import "errors"

var (
	ErrURLNotFound     = errors.New("url not found")
	ErrURLExpired      = errors.New("url has expired")
	ErrInvalidURL      = errors.New("invalid url format")
	ErrShortCodeExists = errors.New("short code already exists")
	ErrInvalidCode     = errors.New("invalid short code format")
)
