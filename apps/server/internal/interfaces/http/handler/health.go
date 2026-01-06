package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/lelemon/server/internal/domain/repository"
)

var (
	// Version is set at build time
	Version   = "dev"
	BuildTime = "unknown"
	startTime = time.Now()
)

// HealthHandler handles health check requests
type HealthHandler struct {
	store repository.Store
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(store repository.Store) *HealthHandler {
	return &HealthHandler{store: store}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string          `json:"status"`
	Version  string          `json:"version"`
	Uptime   string          `json:"uptime"`
	Checks   HealthChecks    `json:"checks"`
	System   *SystemInfo     `json:"system,omitempty"`
}

// HealthChecks contains individual health check results
type HealthChecks struct {
	Database CheckResult `json:"database"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SystemInfo contains system information (only in verbose mode)
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"goroutines"`
	NumCPU       int    `json:"cpus"`
	MemoryMB     uint64 `json:"memory_mb"`
}

// Handle processes GET /health requests
// Query params:
//   - verbose=true: include system information
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	verbose := r.URL.Query().Get("verbose") == "true"

	resp := HealthResponse{
		Status:  "ok",
		Version: Version,
		Uptime:  time.Since(startTime).Round(time.Second).String(),
		Checks: HealthChecks{
			Database: h.checkDatabase(r),
		},
	}

	// Check if any component is unhealthy
	if resp.Checks.Database.Status != "ok" {
		resp.Status = "degraded"
	}

	// Add system info if verbose
	if verbose {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		resp.System = &SystemInfo{
			GoVersion:    runtime.Version(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCPU:       runtime.NumCPU(),
			MemoryMB:     m.Alloc / 1024 / 1024,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if resp.Status != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(resp)
}

// checkDatabase checks the database connection
func (h *HealthHandler) checkDatabase(r *http.Request) CheckResult {
	start := time.Now()
	err := h.store.Ping(r.Context())
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:  "error",
			Latency: latency.Round(time.Millisecond).String(),
			Error:   err.Error(),
		}
	}

	return CheckResult{
		Status:  "ok",
		Latency: latency.Round(time.Millisecond).String(),
	}
}

// LivenessHandler handles GET /health/live (Kubernetes liveness probe)
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ReadinessHandler handles GET /health/ready (Kubernetes readiness probe)
func (h *HealthHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
