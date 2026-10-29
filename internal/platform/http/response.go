package http

import (
	"encoding/json"
	stdhttp "net/http"
)

// Коды ошибок из контракта (Error.code).
const (
	CodeValidationFailed = "validation_failed"
	CodeFeatureDisabled  = "feature_disabled"
	CodeNotFound         = "not_found"
	CodeConflict         = "conflict"
	CodeRateLimited      = "rate_limited"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeInternal         = "internal_error"
)

// ErrorResponse — тело ошибки по контракту.
type ErrorResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// WriteJSON пишет ответ с кодом 200 и телом.
func WriteJSON(w stdhttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// WriteNoContent пишет 204.
func WriteNoContent(w stdhttp.ResponseWriter) {
	w.WriteHeader(stdhttp.StatusNoContent)
}

// WriteError пишет ответ ошибки по контракту с request_id из writer'а.
func WriteError(w stdhttp.ResponseWriter, status int, code, message string, details map[string]any) {
	resp := ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		RequestID: ExtractRequestID(w), // заполняется middleware RequestID
	}
	WriteJSON(w, status, resp)
}

// requestIDer — контракт writer'а, прокидывающего request_id (см. middleware).
type requestIDer interface {
	RequestID() string
}

// ExtractRequestID достаёт request_id из response writer'а, если он его знает.
func ExtractRequestID(w stdhttp.ResponseWriter) string {
	if rw, ok := w.(requestIDer); ok {
		return rw.RequestID()
	}

	return ""
}
