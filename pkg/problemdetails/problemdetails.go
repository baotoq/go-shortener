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

// problemDetailBody is a non-error struct used for JSON serialization in the
// error handler. go-zero's doHandleError writes plaintext if the returned body
// implements error, so we need a plain struct for proper JSON output.
type problemDetailBody struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail"`
	Errors []FieldError `json:"errors,omitempty"`
}

// Error implements the error interface so ProblemDetail can be returned as an error
// from logic layers and detected by the error handler for proper RFC 7807 responses.
func (p *ProblemDetail) Error() string {
	return fmt.Sprintf("%s: %s", p.Title, p.Detail)
}

// Body returns a non-error struct copy suitable for JSON serialization in go-zero's
// error handler (which writes plaintext for error types).
func (p *ProblemDetail) Body() interface{} {
	return problemDetailBody{
		Type:   p.Type,
		Title:  p.Title,
		Status: p.Status,
		Detail: p.Detail,
		Errors: p.Errors,
	}
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
