package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/PenguinTech/MarchProxy/proxy-ailb/internal/providers"
)

// OpenAI-compatible model list response types.

type modelsResponse struct {
	Object string         `json:"object"`
	Data   []modelObject  `json:"data"`
}

type modelObject struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Created  int64  `json:"created"`
	OwnedBy  string `json:"owned_by"`
	Provider string `json:"provider,omitempty"`
}

// ModelsHandler handles GET /v1/models (OpenAI-compatible model listing).
type ModelsHandler struct {
	registry *providers.Registry
}

// NewModelsHandler creates a models listing handler.
func NewModelsHandler(reg *providers.Registry) *ModelsHandler {
	return &ModelsHandler{registry: reg}
}

func (h *ModelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "invalid_request")
		return
	}

	var allModels []modelObject
	for _, p := range h.registry.ListAll() {
		models, err := p.Models(r.Context())
		if err != nil {
			slog.Warn("failed to list models", "provider", p.Name(), "error", err)
			continue
		}
		for _, m := range models {
			allModels = append(allModels, modelObject{
				ID:       m.ID,
				Object:   "model",
				Created:  m.Created,
				OwnedBy:  m.OwnedBy,
				Provider: m.Provider,
			})
		}
	}

	resp := modelsResponse{
		Object: "list",
		Data:   allModels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
