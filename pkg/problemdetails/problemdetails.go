package problemdetails

import "fmt"

// Common problem types
const (
	TypeInvalidURL         = "invalid-url"
	TypeNotFound           = "not-found"
	TypeRateLimitExceeded  = "rate-limit-exceeded"
	TypeInternalError      = "internal-error"
	TypeInvalidRequest     = "invalid-request"
)

// ProblemDetail represents an RFC 7807 Problem Details response
type ProblemDetail struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// New creates a new ProblemDetail with the given parameters
func New(status int, problemType, title, detail string) *ProblemDetail {
	return &ProblemDetail{
		Type:   fmt.Sprintf("https://api.example.com/problems/%s", problemType),
		Title:  title,
		Status: status,
		Detail: detail,
	}
}
