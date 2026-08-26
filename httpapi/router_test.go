package httpapi

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/service"
	"example.com/marine-farm-environment-service/store"
)

// testWebFS exposes the real web directory so the router tests exercise the
// exact assets shipped in production (main.go embeds the same tree).
func testWebFS(t *testing.T) fs.FS {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "web", "index.html")); err != nil {
		t.Fatalf("web assets missing: %v", err)
	}
	return os.DirFS(root)
}

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	cfg.SamplePeriod = time.Second
	cfg.SamplePeriodTolerance = time.Hour
	cfg.RestoreCheckInterval = time.Second
	st := store.NewMemoryStore()
	svc := service.New(cfg, st)
	router := NewRouter(cfg, st, svc, testWebFS(t))
	return router
}

func doJSON(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operator", "tester")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) (map[string]interface{}, int) {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rec.Body.String())
	}
	return out, rec.Code
}

func TestHealthz(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", rec.Code)
	}
	rec2 := doJSON(t, h, http.MethodGet, "/api/healthz", nil)
	if rec2.Code != http.StatusOK {
		t.Fatalf("api healthz status = %d", rec2.Code)
	}
}

func TestIndexServed(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("海洋牧场养殖环境监测")) {
		t.Fatal("index must contain the app title")
	}
}

func TestSPAFallback(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	req := httptest.NewRequest(http.MethodGet, "/warnings", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("spa fallback status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`id="app"`)) {
		t.Fatal("spa fallback must serve the app shell")
	}
}

func TestStaticAssetsServed(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	for _, path := range []string{"/style.css", "/app.js", "/api.js", "/enums.js", "/components/zone_card.js", "/hooks/use_zones.js", "/pages/overview.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
	}
}

func TestFullBusinessChainViaHTTP(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()

	// 1. Create zone.
	rec := doJSON(t, h, http.MethodPost, "/api/zones", map[string]interface{}{
		"name": "测试区", "area": 100, "stock": 50000,
	})
	zoneBody, code := decodeResponse(t, rec)
	if code != http.StatusCreated {
		t.Fatalf("create zone failed: %d %s", code, rec.Body.String())
	}
	zone := zoneBody["data"].(map[string]interface{})
	zoneID := zone["id"].(string)

	// 2. Create two buoys for cross validation.
	buoys := []string{}
	for _, name := range []string{"A浮标", "B浮标"} {
		rec := doJSON(t, h, http.MethodPost, "/api/buoys", map[string]interface{}{
			"zone_id": zoneID, "name": name,
		})
		body, code := decodeResponse(t, rec)
		if code != http.StatusCreated {
			t.Fatalf("create buoy failed: %d %s", code, rec.Body.String())
		}
		buoys = append(buoys, body["data"].(map[string]interface{})["id"].(string))
	}

	now := time.Now().UTC()
	// 3. Neighbour reports normal DO two minutes ago.
	rec = doJSON(t, h, http.MethodPost, "/api/buoys/"+buoys[1]+"/samples", map[string]interface{}{
		"do": 6.1, "temperature": 24, "salinity": 31, "ph": 8.1, "ammonia": 0.05,
		"timestamp": now.Add(-2 * time.Minute).Format(time.RFC3339),
	})
	if body, code := decodeResponse(t, rec); code != http.StatusCreated {
		t.Fatalf("neighbour sample failed: %d %s", code, rec.Body.String())
	} else if body["data"].(map[string]interface{})["aeration_issued"].(bool) {
		t.Fatal("normal sample must not issue aeration")
	}

	// 4. Danger report from buoy A -> cross-checked pending, no aeration.
	rec = doJSON(t, h, http.MethodPost, "/api/buoys/"+buoys[0]+"/samples", map[string]interface{}{
		"do": 2.4, "temperature": 24, "salinity": 31, "ph": 8.1, "ammonia": 0.05,
		"timestamp": now.Format(time.RFC3339),
	})
	dangerBody, code := decodeResponse(t, rec)
	if code != http.StatusCreated {
		t.Fatalf("danger sample failed: %d %s", code, rec.Body.String())
	}
	danger := dangerBody["data"].(map[string]interface{})
	if danger["aeration_issued"].(bool) {
		t.Fatal("cross-checked pending danger must not issue aeration")
	}
	cc := danger["cross_check"].(map[string]interface{})
	if !cc["contradicted"].(bool) {
		t.Fatalf("cross check should be contradicted: %v", cc)
	}
	pendingID := danger["warnings_created"].([]interface{})[0].(map[string]interface{})["id"].(string)

	// 5. Verify the pending warning -> aeration starts, zone aerating.
	rec = doJSON(t, h, http.MethodPost, "/api/warnings/"+pendingID+"/verify", nil)
	if body, code := decodeResponse(t, rec); code != http.StatusOK {
		t.Fatalf("verify failed: %d %s", code, rec.Body.String())
	} else if body["data"].(map[string]interface{})["status"].(string) != "confirmed" {
		t.Fatalf("verify must confirm: %s", rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/api/zones/"+zoneID, nil)
	zoneAfter, code := decodeResponse(t, rec)
	if code != http.StatusOK {
		t.Fatalf("get zone failed: %d", code)
	}
	if zoneAfter["data"].(map[string]interface{})["status"].(string) != "aerating" {
		t.Fatalf("zone should be aerating after verify: %s", rec.Body.String())
	}

	// 6. Restore before eligibility -> 409.
	rec = doJSON(t, h, http.MethodPost, "/api/zones/"+zoneID+"/restore", nil)
	if _, code := decodeResponse(t, rec); code != http.StatusConflict {
		t.Fatalf("restore before eligibility must conflict, got %d", code)
	}

	// 7. Farm log with abnormal death.
	rec = doJSON(t, h, http.MethodPost, "/api/logs", map[string]interface{}{
		"zone_id": zoneID, "date": "2026-08-25", "feed_amount": 500, "death_count": 800,
	})
	logBody, code := decodeResponse(t, rec)
	if code != http.StatusCreated {
		t.Fatalf("farm log failed: %d %s", code, rec.Body.String())
	}
	if !logBody["data"].(map[string]interface{})["death_abnormal"].(bool) {
		t.Fatal("800 deaths must be flagged abnormal")
	}

	// 8. Overview aggregates the chain.
	rec = doJSON(t, h, http.MethodGet, "/api/overview", nil)
	ov, code := decodeResponse(t, rec)
	if code != http.StatusOK {
		t.Fatalf("overview failed: %d", code)
	}
	totals := ov["data"].(map[string]interface{})["totals"].(map[string]interface{})
	if int(totals["open_warning_count"].(float64)) < 1 {
		t.Fatalf("expected open warnings, got %v", totals)
	}

	// 9. Warnings list filters.
	rec = doJSON(t, h, http.MethodGet, "/api/warnings?zone_id="+zoneID+"&status=pending", nil)
	wlist, code := decodeResponse(t, rec)
	if code != http.StatusOK {
		t.Fatalf("warnings list failed: %d", code)
	}
	if arr := wlist["data"].([]interface{}); len(arr) != 0 {
		t.Fatalf("no pending warnings expected after verify, got %d", len(arr))
	}

	// 10. Audit trail contains the business actions.
	rec = doJSON(t, h, http.MethodGet, "/api/audit?limit=50", nil)
	auditBody, code := decodeResponse(t, rec)
	if code != http.StatusOK {
		t.Fatalf("audit failed: %d", code)
	}
	entries := auditBody["data"].([]interface{})
	if len(entries) < 5 {
		t.Fatalf("expected a rich audit trail, got %d entries", len(entries))
	}
}

func TestValidationErrors(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()
	rec := doJSON(t, h, http.MethodPost, "/api/zones", map[string]interface{}{"name": ""})
	_, code := decodeResponse(t, rec)
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d", code)
	}
	rec = doJSON(t, h, http.MethodPost, "/api/zones/zone_999/aerate", nil)
	if _, code := decodeResponse(t, rec); code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing zone, got %d", code)
	}
}

// ensureWebFSIsEmbedded guards against accidental removal of the web dir.
func TestEmbedHasWebAssets(t *testing.T) {
	if _, err := fs.Stat(testWebFS(t), "web/index.html"); err != nil {
		t.Fatalf("web/index.html missing: %v", err)
	}
	if _, err := fs.Stat(testWebFS(t), "web/components/zone_card.js"); err != nil {
		t.Fatalf("web/components missing: %v", err)
	}
}
