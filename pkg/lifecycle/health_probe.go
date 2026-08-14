package lifecycle

import (
	"context"
	"io"
	"net/http"
	"time"
)

const HealthProbeTimeout = 5 * time.Second

// RunHealthProbe GETs targetURL and returns true when status is 200.
// Uses client + context timeouts and always drains/closes the response body.
func RunHealthProbe(targetURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), HealthProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, http.NoBody)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: HealthProbeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	return resp.StatusCode == http.StatusOK
}
