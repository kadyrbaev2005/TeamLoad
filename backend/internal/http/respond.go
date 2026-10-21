// Package http is the TeamLoad REST API: router, handlers and middleware.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"teamload/internal/domain"
)

// apiError is the body of every error response.
type apiError struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *apiError) Error() string { return e.Message }

func errValidation(fields domain.FieldErrors) *apiError {
	msg := "Check the highlighted fields."
	if len(fields) == 1 {
		for _, m := range fields {
			msg = m
		}
	}
	return &apiError{Status: http.StatusBadRequest, Code: "validation", Message: msg, Fields: fields}
}

func errField(field, msg string) *apiError {
	return errValidation(domain.FieldErrors{field: msg})
}

func errBadRequest(msg string) *apiError {
	return &apiError{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

func errUnauthorized() *apiError {
	return &apiError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "Please log in to continue."}
}

func errForbidden(msg string) *apiError {
	return &apiError{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

func errNotFound(msg string) *apiError {
	if msg == "" {
		msg = "Not found."
	}
	return &apiError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

func errConflict(code, msg string) *apiError {
	return &apiError{Status: http.StatusConflict, Code: code, Message: msg}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apiError
	if !errors.As(err, &ae) {
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		ae = &apiError{Status: http.StatusInternalServerError, Code: "internal", Message: "Something went wrong. Please try again."}
	}
	writeJSON(w, ae.Status, map[string]any{"error": ae})
}

// decode reads a JSON body (max 64 KB) into v.
func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	if err := dec.Decode(v); err != nil {
		return errBadRequest("The request body is not valid JSON.")
	}
	return nil
}

// handler adapts a func returning an error to http.HandlerFunc.
type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		writeErr(w, r, err)
	}
}
