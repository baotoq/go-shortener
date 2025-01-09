// Code generated by ent, DO NOT EDIT.

package url

import (
	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the url type in the database.
	Label = "url"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldOriginalURL holds the string denoting the original_url field in the database.
	FieldOriginalURL = "original_url"
	// FieldShortenedURL holds the string denoting the shortened_url field in the database.
	FieldShortenedURL = "shortened_url"
	// Table holds the table name of the url in the database.
	Table = "urls"
)

// Columns holds all SQL columns for url fields.
var Columns = []string{
	FieldID,
	FieldOriginalURL,
	FieldShortenedURL,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

// OrderOption defines the ordering options for the Url queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByOriginalURL orders the results by the original_url field.
func ByOriginalURL(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldOriginalURL, opts...).ToFunc()
}

// ByShortenedURL orders the results by the shortened_url field.
func ByShortenedURL(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldShortenedURL, opts...).ToFunc()
}
