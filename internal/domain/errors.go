package domain

import (
	"errors"

	"go-shortener/internal/domain/valueobject"
)

var (
	ErrURLNotFound     = errors.New("url not found")
	ErrURLExpired      = errors.New("url has expired")
	ErrShortCodeExists = errors.New("short code already exists")

	// Re-export value object errors for convenience.
	ErrInvalidURL  = valueobject.ErrInvalidURL
	ErrInvalidCode = valueobject.ErrInvalidCode
)
