package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"example.com/marine-farm-environment-service/config"
	"example.com/marine-farm-environment-service/middleware"
	"example.com/marine-farm-environment-service/service"
	"example.com/marine-farm-environment-service/store"
)

// maxHTTPAuditEntries caps the request audit trail.
const maxHTTPAuditEntries = 20000

// Handler is the minimal contract every API handler implements.
type Handler interface {
	Register(mux *http.ServeMux)
}

// Router builds the full HTTP handler: middleware chain, REST API and the
// embedded SPA static server.
type Router struct {
	cfg      *config.Config
	store    *store.Store
	svc      *service.Services
	webFS    fs.FS
	staticFS fs.FS
}

// NewRouter constructs the router with the embedded web filesystem.
func NewRouter(cfg *config.Config, st *store.Store, svc *service.Services, webFS fs.FS) *Router {
	static, err := fs.Sub(webFS, "web")
	if err != nil {
		slog.Error("httpapi: embed web fs failed", "error", err)
		panic("httpapi: embed web fs: " + err.Error())
	}
	return &Router{cfg: cfg, store: st, svc: svc, webFS: webFS, staticFS: static}
}

// Handler assembles the middleware-wrapped mux.
func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	r.registerAPI(mux)
	r.registerWeb(mux)

	audit := middleware.NewAuditLogger(r.store, maxHTTPAuditEntries,
		"/api/healthz", "/healthz", "/style.css", "/app.js", "/api.js",
		"/components/", "/hooks/", "/pages/", "/favicon")
	chain := middleware.RequestIDMiddleware(
		middleware.PanicRecoveryMiddleware(
			middleware.SecurityHeadersMiddleware(
				middleware.RequestLoggerMiddleware(audit.Wrap(mux)),
			),
		),
	)
	return chain
}

// registerAPI wires every REST endpoint. Routes use Go 1.22+ path
// parameters via http.ServeMux.
func (r *Router) registerAPI(mux *http.ServeMux) {
	health := NewHealthHandler(r.store)
	zones := NewZoneHandler(r.svc)
	buoys := NewBuoyHandler(r.svc)
	samples := NewSampleHandler(r.svc)
	warnings := NewWarningHandler(r.svc)
	aeration := NewAerationHandler(r.svc)
	farmlogs := NewFarmLogHandler(r.svc)
	overview := NewOverviewHandler(r.svc)

	mux.HandleFunc("GET /api/healthz", health.Healthz)
	mux.HandleFunc("GET /healthz", health.Healthz)
	mux.HandleFunc("GET /api/readyz", health.Readyz)
	mux.HandleFunc("GET /readyz", health.Readyz)

	mux.HandleFunc("GET /api/overview", overview.Get)
	mux.HandleFunc("GET /api/audit", overview.Audit)

	mux.HandleFunc("GET /api/zones", zones.List)
	mux.HandleFunc("POST /api/zones", zones.Create)
	mux.HandleFunc("GET /api/zones/{id}", zones.Get)
	mux.HandleFunc("GET /api/zones/{id}/samples", samples.ZoneSamples)

	mux.HandleFunc("GET /api/buoys", buoys.List)
	mux.HandleFunc("POST /api/buoys", buoys.Create)
	mux.HandleFunc("GET /api/buoys/{id}", buoys.Get)
	mux.HandleFunc("POST /api/buoys/{id}/samples", samples.PostSample)

	mux.HandleFunc("GET /api/warnings", warnings.List)
	mux.HandleFunc("POST /api/warnings/{id}/verify", warnings.Verify)
	mux.HandleFunc("POST /api/warnings/{id}/resolve", warnings.Resolve)

	mux.HandleFunc("GET /api/aeration", aeration.List)
	mux.HandleFunc("POST /api/zones/{id}/aerate", aeration.Start)
	mux.HandleFunc("POST /api/zones/{id}/stop-aeration", aeration.Stop)
	mux.HandleFunc("POST /api/zones/{id}/restore", aeration.Restore)
	mux.HandleFunc("POST /api/aeration/{id}/feedback", aeration.Feedback)

	mux.HandleFunc("GET /api/logs", farmlogs.List)
	mux.HandleFunc("POST /api/logs", farmlogs.Create)
	mux.HandleFunc("GET /api/logs/{id}", farmlogs.Get)
}

// registerWeb serves the embedded SPA. Static assets are served directly;
// unknown non-API paths fall back to index.html for client-side routing.
func (r *Router) registerWeb(mux *http.ServeMux) {
	fileServer := http.FileServer(http.FS(r.staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if strings.HasPrefix(path, "/api/") {
			FailPlain(w, req, http.StatusNotFound, "api route not found: "+path)
			return
		}
		// Serve an actual static file when it exists in the embedded FS.
		if path != "/" {
			if f, err := r.staticFS.Open(strings.TrimPrefix(path, "/")); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, req)
				return
			}
		}
		// SPA fallback: always serve the app shell.
		data, err := fs.ReadFile(r.staticFS, "index.html")
		if err != nil {
			FailPlain(w, req, http.StatusInternalServerError, "app shell missing")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
