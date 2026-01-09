package handler

import (
	"encoding/json"
	"net/http"
)

// FeaturesConfig defines what features are available in this server instance.
// This is duplicated from the http package to avoid import cycles.
type FeaturesConfig struct {
	Edition  string          `json:"edition"` // "community" or "enterprise"
	Features map[string]bool `json:"features"`
}

// FeaturesHandler handles the /api/v1/features endpoint.
// This endpoint allows the frontend to detect which features are available.
type FeaturesHandler struct {
	config *FeaturesConfig
}

// NewFeaturesHandler creates a new features handler with the given config.
// Pass nil to use default community features.
func NewFeaturesHandler(config *FeaturesConfig) *FeaturesHandler {
	if config == nil {
		config = DefaultFeaturesConfig()
	}
	return &FeaturesHandler{config: config}
}

// DefaultFeaturesConfig returns the default features for community edition.
func DefaultFeaturesConfig() *FeaturesConfig {
	return &FeaturesConfig{
		Edition: "community",
		Features: map[string]bool{
			"organizations": false,
			"rbac":          false,
			"billing":       false,
			"sso":           false,
		},
	}
}

// Handle returns the server's feature configuration.
// GET /api/v1/features
func (h *FeaturesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.config)
}
