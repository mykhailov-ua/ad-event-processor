package postback

import (
	"fmt"
	"io"
	"net/http"
)

// DispatchHTTPError is returned when a provider receives a non-2xx HTTP response.
type DispatchHTTPError struct {
	StatusCode int
	Body       string
}

func (e *DispatchHTTPError) Error() string {
	if e == nil {
		return "dispatch http error"
	}
	return fmt.Sprintf("unexpected status code %d: %s", e.StatusCode, e.Body)
}

// Permanent reports client errors (4xx) that must not be retried.
func (e *DispatchHTTPError) Permanent() bool {
	return e != nil && e.StatusCode >= 400 && e.StatusCode < 500
}

func checkHTTPResponse(resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return &DispatchHTTPError{StatusCode: resp.StatusCode, Body: string(body)}
}
