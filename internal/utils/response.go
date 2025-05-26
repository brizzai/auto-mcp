package utils

import (
	"encoding/json"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// writeJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response
func WriteError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": message,
	}); err != nil {
		logger.Error("Failed to encode error response", zap.Error(err))
	}
}
