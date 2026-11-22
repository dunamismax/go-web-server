package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dunamismax/go-web-server/internal/view"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) error {
	return render(w, r, http.StatusOK, view.Home(s.page(r, "")))
}

// healthz is the liveness probe: the process is up and serving. It checks
// nothing else on purpose, so a database outage does not get the app
// restarted in a loop.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// readyz is the readiness probe: the app can reach its database and should
// receive traffic.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		s.log.WarnContext(r.Context(), "readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "database": "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
