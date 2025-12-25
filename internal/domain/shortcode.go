package domain

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"
)

const (
	DefaultShortCodeLength = 6
	MinCustomCodeLength    = 3
	MaxCustomCodeLength    = 20
)

var shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ShortCode is a value object representing a URL short code.
// It is immutable and validated on creation.
type ShortCode struct {
	value string
}

// NewShortCode creates a new ShortCode from a string, validating the format.
func NewShortCode(code string) (ShortCode, error) {
	if err := validateShortCode(code); err != nil {
		return ShortCode{}, err
	}
	return ShortCode{value: code}, nil
}

// GenerateShortCode creates a new random ShortCode of the specified length.
func GenerateShortCode(length int) (ShortCode, error) {
	if length <= 0 {
		length = DefaultShortCodeLength
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return ShortCode{}, err
	}

	code := base64.URLEncoding.EncodeToString(bytes)
	code = strings.TrimRight(code, "=")
	if len(code) > length {
		code = code[:length]
	}

	return ShortCode{value: code}, nil
}

// String returns the string representation of the ShortCode.
func (s ShortCode) String() string {
	return s.value
}

// IsEmpty returns true if the ShortCode is empty.
func (s ShortCode) IsEmpty() bool {
	return s.value == ""
}

// Equals compares two ShortCodes for equality.
func (s ShortCode) Equals(other ShortCode) bool {
	return s.value == other.value
}

func validateShortCode(code string) error {
	if code == "" {
		return ErrInvalidCode
	}

	if len(code) < MinCustomCodeLength || len(code) > MaxCustomCodeLength {
		return ErrInvalidCode
	}

	if !shortCodeRegex.MatchString(code) {
		return ErrInvalidCode
	}

	return nil
}
