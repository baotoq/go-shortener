package domain

import (
	"time"

	"go-shortener/internal/domain/event"
)

// Compile-time interface checks
var (
	_ AggregateRoot = (*URL)(nil)
	_ AggregateRoot = (*DeletedURLAggregate)(nil)
)

// URL is the aggregate root representing a shortened URL.
// It encapsulates all business logic related to URL shortening.
type URL struct {
	id          int64
	shortCode   ShortCode
	originalURL OriginalURL
	clickCount  int64
	expiresAt   *time.Time
	createdAt   time.Time
	updatedAt   time.Time

	// events holds domain events raised by this aggregate.
	events []event.Event
}

// NewURL creates a new URL entity with the given short code and original URL.
// It raises a URLCreated event.
func NewURL(shortCode ShortCode, originalURL OriginalURL, expiresAt *time.Time) *URL {
	now := time.Now().UTC()
	u := &URL{
		shortCode:   shortCode,
		originalURL: originalURL,
		expiresAt:   expiresAt,
		createdAt:   now,
		updatedAt:   now,
		events:      make([]event.Event, 0),
	}
	u.addEvent(event.NewURLCreated(shortCode.String(), originalURL.String(), expiresAt))
	return u
}

// ReconstructURL reconstructs a URL entity from persistence.
// This is used by the repository to recreate the entity from stored data.
func ReconstructURL(
	id int64,
	shortCode ShortCode,
	originalURL OriginalURL,
	clickCount int64,
	expiresAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *URL {
	return &URL{
		id:          id,
		shortCode:   shortCode,
		originalURL: originalURL,
		clickCount:  clickCount,
		expiresAt:   expiresAt,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// ID returns the URL's unique identifier.
func (u *URL) ID() int64 {
	return u.id
}

// ShortCode returns the URL's short code.
func (u *URL) ShortCode() ShortCode {
	return u.shortCode
}

// OriginalURL returns the original URL.
func (u *URL) OriginalURL() OriginalURL {
	return u.originalURL
}

// ClickCount returns the number of times this URL has been accessed.
func (u *URL) ClickCount() int64 {
	return u.clickCount
}

// ExpiresAt returns the expiration time of the URL, or nil if it never expires.
func (u *URL) ExpiresAt() *time.Time {
	return u.expiresAt
}

// CreatedAt returns when the URL was created.
func (u *URL) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns when the URL was last updated.
func (u *URL) UpdatedAt() time.Time {
	return u.updatedAt
}

// IsExpired checks if the URL has expired.
func (u *URL) IsExpired() bool {
	if u.expiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.expiresAt)
}

// CanRedirect checks if the URL can be used for redirection.
// Returns an error if the URL is expired.
func (u *URL) CanRedirect() error {
	if u.IsExpired() {
		return ErrURLExpired
	}
	return nil
}

// RecordClick increments the click count and updates the timestamp.
// It raises a URLClicked event and optionally a ClickMilestoneReached event.
func (u *URL) RecordClick(userAgent, ipAddress, referrer string) {
	previousCount := u.clickCount
	u.clickCount++
	u.updatedAt = time.Now().UTC()

	// Raise URLClicked event
	u.addEvent(event.NewURLClicked(u.shortCode.String(), u.clickCount, userAgent, ipAddress, referrer))

	// Check for milestone
	if milestone := event.CheckMilestone(previousCount, u.clickCount); milestone > 0 {
		u.addEvent(event.NewClickMilestoneReached(u.shortCode.String(), milestone, u.clickCount))
	}
}

// Redirect returns the original URL string if the URL is valid for redirection.
// It also records the click with the provided context.
func (u *URL) Redirect(userAgent, ipAddress, referrer string) (string, error) {
	if err := u.CanRedirect(); err != nil {
		return "", err
	}
	u.RecordClick(userAgent, ipAddress, referrer)
	return u.originalURL.String(), nil
}

// SetID sets the URL's ID. This is typically called by the repository after persistence.
func (u *URL) SetID(id int64) {
	u.id = id
}

// HasExpiration returns true if the URL has an expiration date set.
func (u *URL) HasExpiration() bool {
	return u.expiresAt != nil
}

// addEvent adds a domain event to the aggregate.
func (u *URL) addEvent(e event.Event) {
	u.events = append(u.events, e)
}

// Events returns all uncommitted domain events.
func (u *URL) Events() []event.Event {
	return u.events
}

// ClearEvents clears all domain events after they have been dispatched.
func (u *URL) ClearEvents() {
	u.events = make([]event.Event, 0)
}

// DeletedURLAggregate is a minimal aggregate for raising URLDeleted events.
type DeletedURLAggregate struct {
	events []event.Event
}

// NewDeletedURLAggregate creates a new aggregate with a URLDeleted event.
func NewDeletedURLAggregate(shortCode string) *DeletedURLAggregate {
	agg := &DeletedURLAggregate{
		events: make([]event.Event, 0, 1),
	}
	agg.events = append(agg.events, event.NewURLDeleted(shortCode))
	return agg
}

// Events returns all uncommitted domain events.
func (a *DeletedURLAggregate) Events() []event.Event {
	return a.events
}

// ClearEvents clears all domain events.
func (a *DeletedURLAggregate) ClearEvents() {
	a.events = make([]event.Event, 0)
}
