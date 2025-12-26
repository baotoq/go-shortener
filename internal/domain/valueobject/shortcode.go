package valueobject

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	if err := validation.Validate(code,
		validation.Required.Error("short code is required"),
		validation.Length(MinCustomCodeLength, MaxCustomCodeLength).Error("short code must be 3-20 characters"),
		validation.Match(shortCodeRegex).Error("short code must contain only alphanumeric characters, underscores, and hyphens"),
	); err != nil {
		return ShortCode{}, ErrInvalidCode
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

