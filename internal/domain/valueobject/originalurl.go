package valueobject

import (
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// OriginalURL is a value object representing the original URL to be shortened.
// It is immutable and validated on creation.
type OriginalURL struct {
	value  string
	parsed *url.URL
}

// NewOriginalURL creates a new OriginalURL from a string, validating the format.
func NewOriginalURL(rawURL string) (OriginalURL, error) {
	if err := validation.Validate(rawURL,
		validation.Required.Error("URL is required"),
		is.URL.Error("invalid URL format"),
	); err != nil {
		return OriginalURL{}, ErrInvalidURL
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return OriginalURL{}, ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return OriginalURL{}, ErrInvalidURL
	}

	if parsed.Host == "" {
		return OriginalURL{}, ErrInvalidURL
	}

	return OriginalURL{
		value:  rawURL,
		parsed: parsed,
	}, nil
}

// String returns the string representation of the OriginalURL.
func (o OriginalURL) String() string {
	return o.value
}

// Host returns the host portion of the URL.
func (o OriginalURL) Host() string {
	if o.parsed == nil {
		return ""
	}
	return o.parsed.Host
}

// Scheme returns the scheme (http or https) of the URL.
func (o OriginalURL) Scheme() string {
	if o.parsed == nil {
		return ""
	}
	return o.parsed.Scheme
}

// IsEmpty returns true if the OriginalURL is empty.
func (o OriginalURL) IsEmpty() bool {
	return o.value == ""
}

// Equals compares two OriginalURLs for equality.
func (o OriginalURL) Equals(other OriginalURL) bool {
	return o.value == other.value
}
