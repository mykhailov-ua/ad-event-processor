package payment

import (
	"log/slog"
	"net/http"
	"strings"
)

func WriteHTMXError(w http.ResponseWriter, r *http.Request, err error, logAttrs ...any) {
	status, code, message := MapHTMXError(err)
	if status >= StatusFailed {
		attrs := append([]any{slog.String("error", err.Error())}, logAttrs...)
		slog.Error("payment htmx request failed", attrs...)
	}
	WriteHTMX(w, r, status, code, message)
}

func WriteHTMX(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Payment-Error-Code", code)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(htmxErrorBody(code, message)))
}

func WriteHTMXOK(w http.ResponseWriter, r *http.Request, fragment string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if fragment == "" {
		fragment = `<div id="payment-ok" data-status="ok"></div>`
	}
	_, _ = w.Write([]byte(fragment))
}

func htmxErrorBody(code, message string) string {
	var b strings.Builder
	b.WriteString(`<div id="payment-error" data-code="`)
	b.WriteString(code)
	b.WriteString(`">`)
	b.WriteString(message)
	b.WriteString(`</div>`)
	return b.String()
}
