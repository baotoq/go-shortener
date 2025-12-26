package domain

import (
	"go-shortener/internal/domain/valueobject"
)

// Re-export value object types for convenience.
// This allows consumers to import from domain package directly.
type (
	ShortCode   = valueobject.ShortCode
	OriginalURL = valueobject.OriginalURL
)

// Re-export value object constructors.
var (
	NewShortCode      = valueobject.NewShortCode
	GenerateShortCode = valueobject.GenerateShortCode
	NewOriginalURL    = valueobject.NewOriginalURL
)

// Re-export value object constants.
const (
	DefaultShortCodeLength = valueobject.DefaultShortCodeLength
	MinCustomCodeLength    = valueobject.MinCustomCodeLength
	MaxCustomCodeLength    = valueobject.MaxCustomCodeLength
)
