package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMalformedJSONReturns400 pins the decoding contract: a malformed body
// must be rejected with 400, not escalated to 500.
func TestMalformedJSONReturns400(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/zones", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON status = %d, want 400", rec.Code)
	}
}

// TestMissingFarmLogReturns404 pins the not-found contract for farm logs.
func TestMissingFarmLogReturns404(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodGet, "/api/logs/farmlog_does_not_exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing farm log status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

// TestFarmLogValidationReturns400 pins the validation contract: an invalid
// date must be rejected with 400, not escalated to 500.
func TestFarmLogValidationReturns400(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodPost, "/api/zones", map[string]interface{}{
		"name": "东区", "area": 120, "stock": 50000,
	})
	var created map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	data, _ := created["data"].(map[string]interface{})
	zoneID, _ := data["id"].(string)

	rec = doJSON(t, h, http.MethodPost, "/api/logs", map[string]interface{}{
		"zone_id": zoneID, "date": "2026-13-99", "feed_amount": 10, "death_count": 0,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid farm log status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
