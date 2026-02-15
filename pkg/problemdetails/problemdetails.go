package problemdetails

import "fmt"

const (
	TypeInvalidURL        = "invalid-url"
	TypeNotFound          = "not-found"
	TypeRateLimitExceeded = "rate-limit-exceeded"
	TypeInternalError     = "internal-error"
	TypeValidationError   = "validation-error"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ProblemDetail struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail"`
	Errors []FieldError `json:"errors,omitempty"`
}

func New(status int, problemType, title, detail string) *ProblemDetail {
	return &ProblemDetail{
		Type:   fmt.Sprintf("https://api.example.com/problems/%s", problemType),
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

func NewValidation(errors []FieldError) *ProblemDetail {
	return &ProblemDetail{
		Type:   fmt.Sprintf("https://api.example.com/problems/%s", TypeValidationError),
		Title:  "Validation Failed",
		Status: 400,
		Detail: "Request validation failed",
		Errors: errors,
	}
}
