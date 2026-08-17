// Package control implements control support for BidShard.
package control

import (
	"context"
	"net/http"
	"os"
)

func ProbeHealth(args []string) bool {
	if len(args) > 2 && args[1] == "--health-probe" {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, args[2], http.NoBody)
		if err != nil {
			os.Exit(1)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			os.Exit(1)
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			os.Exit(1)
		}
		_ = resp.Body.Close()
		os.Exit(0)
	}
	return false
}
