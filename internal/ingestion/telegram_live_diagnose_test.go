package ingestion

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTelegramClickLiveDiagnose(t *testing.T) {
	if os.Getenv("TG_LIVE") == "" {
		t.Skip("set TG_LIVE=1")
	}
	base := os.Getenv("TG_LIVE_BASE")
	if base == "" {
		base = "http://127.0.0.1:8181"
	}
	campaign := os.Getenv("TG_SOAK_CAMPAIGN_ID")
	if campaign == "" {
		campaign = "00000000-0000-0000-0000-000000000001"
	}
	token := os.Getenv("TG_SOAK_BRIDGE_TOKEN")
	if token == "" {
		token = "token_abc123_"
	}

	t.Run("net/http keep-alive", func(t *testing.T) {
		client := &http.Client{Timeout: 2 * time.Second}
		t.Log(probeTelegramClick(client, base, campaign, token, 40))
	})
	t.Run("net/http no keep-alive", func(t *testing.T) {
		client := &http.Client{
			Timeout:   2 * time.Second,
			Transport: &http.Transport{DisableKeepAlives: true},
		}
		t.Log(probeTelegramClick(client, base, campaign, token, 40))
	})
}

func probeTelegramClick(client *http.Client, base, campaign, token string, n int) map[string]int {
	kinds := map[string]int{}
	for i := range n {
		clickID := fmt.Sprintf("00000000-0000-4000-8000-%012x", i)
		url := fmt.Sprintf("%s/tg/click?campaign_id=%s&click_id=%s&bridge_token=%s", base, campaign, clickID, token)
		resp, err := client.Get(url)
		if err != nil {
			kinds["transport_error"]++
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		_ = resp.Body.Close()
		kind := classifyTelegramLiveResponse(resp.StatusCode, resp.Header.Get("Content-Length"), body)
		kinds[kind]++
	}
	return kinds
}

func classifyTelegramLiveResponse(status int, contentLen string, body []byte) string {
	switch status {
	case 302:
		return "redirect_302"
	case 404:
		if string(body) == "Not Found" {
			return "tg_no_landing_404"
		}
		return fmt.Sprintf("other_404_body_%q", string(body))
	case 400:
		if contentLen == "0" || len(body) == 0 {
			return "http_parse_400"
		}
		if strings.Contains(string(body), "Invalid Request") {
			return "tg_handler_400"
		}
		return fmt.Sprintf("other_400_body_%q", string(body))
	case 405:
		return "route_missing_405"
	default:
		return fmt.Sprintf("status_%d", status)
	}
}
