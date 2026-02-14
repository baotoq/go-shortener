package http

import (
	"encoding/json"
	"net/http"

	"go-shortener/pkg/problemdetails"
)

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeProblem writes an RFC 7807 Problem Details response
func writeProblem(w http.ResponseWriter, problem *problemdetails.ProblemDetail) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.Status)
	json.NewEncoder(w).Encode(problem)
}
