package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestFeedbackHandlerAppliesStarted pins the HTTP contract: the feedback
// endpoint must forward the requested feedback, not a fixed acknowledged.
func TestFeedbackHandlerAppliesStarted(t *testing.T) {
	router := newTestRouter(t)
	h := router.Handler()

	rec := doJSON(t, h, http.MethodPost, "/api/zones", map[string]interface{}{
		"name": "东区", "area": 120, "stock": 50000,
	})
	var created map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	data, _ := created["data"].(map[string]interface{})
	zoneID, _ := data["id"].(string)

	rec = doJSON(t, h, http.MethodPost, "/api/zones/"+zoneID+"/aerate", nil)
	var aerated map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &aerated)
	adata, _ := aerated["data"].(map[string]interface{})
	aerID, _ := adata["id"].(string)

	rec = doJSON(t, h, http.MethodPost, "/api/aeration/"+aerID+"/feedback", map[string]interface{}{
		"feedback": "started",
	})
	var fb map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &fb)
	fdata, _ := fb["data"].(map[string]interface{})
	status, _ := fdata["status"].(string)
	if status != "running" {
		t.Fatalf("aeration status = %q, want running (feedback not applied)", status)
	}
}
