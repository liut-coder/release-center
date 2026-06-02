package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	Details   map[string]any `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string, details map[string]any) {
	JSON(w, status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: RequestID(r.Context()),
		Details:   details,
	})
}
