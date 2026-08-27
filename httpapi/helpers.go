package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/middleware"
)

// maxJSONBodyBytes caps request bodies at 1 MiB so an oversized payload is
// rejected with a 400 instead of exhausting memory.
const maxJSONBodyBytes = 1 << 20

// pathValue returns a path parameter from a Go 1.22+ mux route.
func pathValue(r *http.Request, key string) string {
	return r.PathValue(key)
}

// decodeJSON parses exactly one JSON object from the request body. It
// rejects malformed JSON, oversized bodies and trailing data after the
// first value.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return bodyError(err)
	}
	// Reject trailing JSON: there must be exactly one value in the body.
	var extra interface{}
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return domain.InvalidInput("request body must contain a single JSON value")
		}
		return bodyError(err)
	}
	return nil
}

// bodyError translates a JSON decoding error into a stable 400-style
// invalid-input domain error so Err() maps it to http.StatusBadRequest.
// Returning a plain fmt.Errorf here would lose the domain code and every
// malformed/oversized body would surface as a 500 "internal" error.
func bodyError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return domain.InvalidInput("request body too large (limit %d bytes)", maxJSONBodyBytes)
	}
	return domain.InvalidInput("malformed JSON body: %v", err)
}

// operatorName returns the operator header or a default.
func operatorName(r *http.Request) string {
	if v := r.Header.Get("X-Operator"); v != "" {
		return v
	}
	return "anonymous"
}

// requestIDFrom returns the trace id from the middleware context.
func requestIDFrom(r *http.Request) string {
	return middleware.RequestID(r.Context())
}

// parseLimitOffset parses and validates the limit/offset query parameters.
// Negative or non-integer values are rejected as invalid input. The limit
// is clamped to the supplied maximum and defaults when absent.
func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (int, int, error) {
	q := r.URL.Query()
	limit := defaultLimit
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, domain.InvalidInput("limit must be a non-negative integer")
		}
		limit = n
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, domain.InvalidInput("offset must be a non-negative integer")
		}
		offset = n
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if limit == 0 {
		limit = defaultLimit
		if limit > maxLimit {
			limit = maxLimit
		}
	}
	return limit, offset, nil
}
