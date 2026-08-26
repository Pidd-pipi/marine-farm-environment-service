package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/marine-farm-environment-service/domain"
	"example.com/marine-farm-environment-service/store"
)

func TestRequestIDMiddleware(t *testing.T) {
	var gotID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := RequestIDMiddleware(inner)

	// Incoming id is honoured.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "trace-abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if gotID != "trace-abc" {
		t.Fatalf("expected incoming trace id, got %s", gotID)
	}
	if rec.Header().Get("X-Request-Id") != "trace-abc" {
		t.Fatal("response must echo the trace id")
	}

	// Generated id when absent.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if gotID == "" || gotID == "trace-abc" {
		t.Fatalf("expected a generated id, got %s", gotID)
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	handler := PanicRecoveryMiddleware(RequestIDMiddleware(inner))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Body.String() == "" {
		t.Fatal("expected a JSON error body")
	}
}

func TestAuditLogger(t *testing.T) {
	st := store.NewMemoryStore()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	audit := NewAuditLogger(st, 1000, "/api/healthz")
	handler := RequestIDMiddleware(PanicRecoveryMiddleware(audit.Wrap(inner)))

	req := httptest.NewRequest(http.MethodPost, "/api/zones", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entries := st.Audit().List(10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Action != domain.AuditHTTPRequest || e.TargetID != "/api/zones" {
		t.Fatalf("unexpected audit entry: %+v", e)
	}
}

func TestAuditLoggerSkips(t *testing.T) {
	st := store.NewMemoryStore()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	audit := NewAuditLogger(st, 1000, "/api/healthz", "/style.css")
	handler := RequestIDMiddleware(PanicRecoveryMiddleware(audit.Wrap(inner)))

	for _, path := range []string{"/api/healthz", "/style.css"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
	if n := st.Audit().Count(); n != 0 {
		t.Fatalf("skipped paths must not be audited, got %d entries", n)
	}
}
