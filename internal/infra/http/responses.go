package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"marginalia/internal/common"
	"net/http"
)

func JsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func WriteError(w http.ResponseWriter, err error) {
	var svcErr common.ServiceError
	if errors.As(err, &svcErr) {
		JsonError(w, svcErr.Reason, svcErr.Code)
		return
	}

	slog.Error("unhandled error in HTTP handler", "error", err)
	JsonError(w, "internal server error", http.StatusInternalServerError)
}
