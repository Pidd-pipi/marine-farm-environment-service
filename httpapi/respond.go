// Package httpapi implements the REST API layer. Handlers translate HTTP
// requests into service calls and render the unified response envelope.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/middleware"
)

// Response is the unified response envelope used by every API endpoint.
//
//	{"code":0,"message":"ok","data":...}
//	{"code":404,"message":"...","error":"not_found","request_id":"..."}
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// OK writes a successful response with the given payload.
func OK(w http.ResponseWriter, r *http.Request, data interface{}) {
	WriteJSON(w, r, http.StatusOK, Response{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: middleware.RequestID(r.Context()),
	})
}

// Created writes a 201 response with the given payload.
func Created(w http.ResponseWriter, r *http.Request, data interface{}) {
	WriteJSON(w, r, http.StatusCreated, Response{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: middleware.RequestID(r.Context()),
	})
}

// Err writes an error response, mapping domain error codes to HTTP status.
func Err(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	errCode := string(domain.CodeInternal)
	var de *domain.Error
	if domain.As(err, &de) {
		switch de.Code {
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeInvalidInput:
			status = http.StatusBadRequest
		case domain.CodeConflict:
			status = http.StatusConflict
		case domain.CodeUnauthorized:
			status = http.StatusUnauthorized
		default:
			status = http.StatusInternalServerError
		}
		errCode = string(de.Code)
	}
	WriteJSON(w, r, status, Response{
		Code:      status,
		Message:   err.Error(),
		Error:     errCode,
		RequestID: middleware.RequestID(r.Context()),
	})
}

// WriteJSON writes a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Warn("httpapi: encode response failed", "error", err, "request_id", middleware.RequestID(r.Context()))
	}
}

// FailPlain writes a plain-text error (used for static-file fallbacks).
func FailPlain(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

// setListHeaders exposes pagination metadata as response headers while the
// body remains a plain JSON array so the existing frontend contract is
// preserved.
func setListHeaders(w http.ResponseWriter, limit, offset, total int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Offset", strconv.Itoa(offset))
}

// paginate returns the requested window of items and the total length. The
// returned slice is always non-nil so JSON encodes it as [].
func paginate[T any](items []T, offset, limit int) ([]T, int) {
	total := len(items)
	if offset >= total {
		return []T{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return items[offset:end], total
}
