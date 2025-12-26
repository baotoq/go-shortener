package domain

//go:generate mockery --name=URLRepository --output=../mocks --outpkg=mocks --with-expecter

import (
	"context"
)

// URLRepository defines the interface for URL persistence operations.
// This interface is defined in the domain layer and implemented in the data layer,
// following the Dependency Inversion Principle.
type URLRepository interface {
	// Save persists a URL entity. If the URL is new (ID == 0), it creates a new record.
	// If the URL exists, it updates the existing record.
	Save(ctx context.Context, url *URL) error

	// FindByShortCode retrieves a URL by its short code.
	// Returns nil if not found.
	FindByShortCode(ctx context.Context, code ShortCode) (*URL, error)

	// Delete removes a URL by its short code.
	Delete(ctx context.Context, code ShortCode) error

	// FindAll retrieves all URLs with pagination.
	// Returns the list of URLs and the total count.
	FindAll(ctx context.Context, page, pageSize int) ([]*URL, int, error)

	// Exists checks if a short code already exists.
	Exists(ctx context.Context, code ShortCode) (bool, error)

	// IncrementClickCount atomically increments the click count for a URL.
	// This is separated from Save for performance optimization (atomic update).
	IncrementClickCount(ctx context.Context, code ShortCode) error
}
