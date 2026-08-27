package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// These tests pin the error-mapping contract: missing business resources
// must surface as 404 and domain conflicts as 409, never as a generic 500.
func TestGetMissingZoneReturns404(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodGet, "/api/zones/zone_does_not_exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing zone status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	_, code := decodeResponse(t, rec)
	if code != http.StatusNotFound {
		t.Fatalf("missing zone code = %d, want 404", code)
	}
}

func TestGetMissingBuoyReturns404(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodGet, "/api/buoys/buoy_does_not_exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing buoy status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	_, code := decodeResponse(t, rec)
	if code != http.StatusNotFound {
		t.Fatalf("missing buoy code = %d, want 404", code)
	}
}

func TestRestoreConflictReturns409(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()

	// Create a normal zone, then ask to restore it: the zone is not
	// aerating, so the restore must be rejected as a domain conflict.
	rec := doJSON(t, h, http.MethodPost, "/api/zones", map[string]interface{}{
		"name": "东区测试", "area": 120, "stock": 50000,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create zone status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create zone: %v", err)
	}
	data, _ := created["data"].(map[string]interface{})
	zoneID, _ := data["id"].(string)
	if zoneID == "" {
		t.Fatalf("created zone has no id: %s", rec.Body.String())
	}

	rec2 := doJSON(t, h, http.MethodPost, "/api/zones/"+zoneID+"/restore", nil)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("restore conflict status = %d, want 409; body=%s", rec2.Code, rec2.Body.String())
	}
	_, code := decodeResponse(t, rec2)
	if code != http.StatusConflict {
		t.Fatalf("restore conflict code = %d, want 409", code)
	}
}
