package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/lelemon/server/pkg/domain/repository"
)

var (
	// Version is set at build time
	Version   = "dev"
	BuildTime = "unknown"
	startTime = time.Now()
)

// HealthHandler handles health check requests
type HealthHandler struct {
	primaryStore   repository.Store
	analyticsStore repository.Store
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(primaryStore, analyticsStore repository.Store) *HealthHandler {
	return &HealthHandler{
		primaryStore:   primaryStore,
		analyticsStore: analyticsStore,
	}
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
	Primary   CheckResult  `json:"primary"`
	Analytics *CheckResult `json:"analytics,omitempty"` // Only shown if different from primary
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

	primaryCheck := h.checkStore(r, h.primaryStore)
	resp := HealthResponse{
		Status:  "ok",
		Version: Version,
		Uptime:  time.Since(startTime).Round(time.Second).String(),
		Checks: HealthChecks{
			Primary: primaryCheck,
		},
	}

	// Check analytics store if different from primary
	if h.analyticsStore != h.primaryStore {
		analyticsCheck := h.checkStore(r, h.analyticsStore)
		resp.Checks.Analytics = &analyticsCheck
		if analyticsCheck.Status != "ok" {
			resp.Status = "degraded"
		}
	}

	// Check if any component is unhealthy
	if primaryCheck.Status != "ok" {
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

// checkStore checks a store's database connection
func (h *HealthHandler) checkStore(r *http.Request, store repository.Store) CheckResult {
	start := time.Now()
	err := store.Ping(r.Context())
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
	// Check primary store
	if err := h.primaryStore.Ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"error":  "primary: " + err.Error(),
		})
		return
	}

	// Check analytics store if different
	if h.analyticsStore != h.primaryStore {
		if err := h.analyticsStore.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"error":  "analytics: " + err.Error(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
