package httpresponse

import (
	"encoding/json"
	"net/http"
)

type ErrorDTO struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDTO `json:"error"`
}

// JSON serializes payload into standard JSON formatting. Ensuring consistent Content-Type headers across all responses permits automated client-side interceptors to rely on predictable payload schemas.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error encapsulates domain error details into a uniform DTO. This standardizes error boundaries and prevents raw internal stack traces or database errors from leaking to external consumers.
func Error(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Error: ErrorDTO{
			Code:    code,
			Message: message,
		},
	}
	JSON(w, status, resp)
}
