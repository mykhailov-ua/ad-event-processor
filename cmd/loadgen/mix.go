package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type mixConfig struct {
	pctOpenRTB    int
	pctTelegram   int
	pctValid      int
	pctFraud      int
	pctInvalid    int
	pctDDoS       int
	pctClickProxy int
}

func defaultMix(mode string, pctBroken, pctGray int) mixConfig {
	switch mode {
	case "business":
		if pctBroken <= 0 {
			pctBroken = 50
		}
		if pctGray <= 0 {
			pctGray = 20
		}
		clean := 100 - pctBroken - pctGray - 5
		if clean < 0 {
			clean = 0
		}
		pctTelegram := 8
		if clean > pctTelegram {
			clean -= pctTelegram
		} else {
			pctTelegram = clean
			clean = 0
		}
		return mixConfig{
			pctOpenRTB:  5,
			pctTelegram: pctTelegram,
			pctValid:    clean,
			pctFraud:    pctGray,
			pctInvalid:  pctBroken / 2,
			pctDDoS:     pctBroken - pctBroken/2,
		}
	case "smoke":
		return mixConfig{pctOpenRTB: 12, pctTelegram: 5, pctValid: 28, pctFraud: 10, pctInvalid: 20, pctDDoS: 15}
	default:
		return mixConfig{pctOpenRTB: 8, pctTelegram: 5, pctValid: 22, pctFraud: 15, pctInvalid: 20, pctDDoS: 15}
	}
}

type runner struct {
	client        *http.Client
	trackers      []string
	edgeURL       string
	oversizeBytes int
	mix           mixConfig
	hist          *histogram
	campaignID    string
	iter          atomic.Uint64
}

func newRunner(trackers []string, edgeURL string, oversize int, mix mixConfig, hist *histogram) *runner {
	return &runner{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        512,
				MaxIdleConnsPerHost: 256,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
			},
		},
		trackers:      trackers,
		edgeURL:       edgeURL,
		oversizeBytes: oversize,
		mix:           mix,
		hist:          hist,
	}
}

func campaignID(iter uint64) string {
	n := (iter % 100) + 1
	return fmt.Sprintf("00000000-0000-0000-0000-%012x", n)
}

func (r *runner) pickCampaign(iter uint64) string {
	if r.campaignID != "" {
		return r.campaignID
	}
	return campaignID(iter)
}

func (r *runner) doOnce() {
	iter := r.iter.Add(1)
	roll := rand.Intn(100)
	base := r.trackers[iter%uint64(len(r.trackers))]

	openrtbEnd := r.mix.pctOpenRTB
	telegramEnd := openrtbEnd + r.mix.pctTelegram
	clickProxyEnd := telegramEnd + r.mix.pctClickProxy
	validEnd := clickProxyEnd + r.mix.pctValid
	fraudEnd := validEnd + r.mix.pctFraud
	invalidEnd := fraudEnd + r.mix.pctInvalid
	ddosEnd := invalidEnd + r.mix.pctDDoS

	switch {
	case roll < openrtbEnd:
		r.postOpenRTBBid(base, iter)
	case roll < telegramEnd:
		r.telegramTraffic(base, iter)
	case roll < clickProxyEnd:
		r.clickProxyTraffic(base, iter)
	case roll < validEnd:
		body := r.validBody(iter)
		r.post(base+"/track", "application/json", body, nil)
	case roll < fraudEnd:
		body := r.fraudBody(iter)
		r.post(base+"/track", "application/json", body, map[string]string{
			"X-Forwarded-For": fraudIP(iter),
		})
	case roll < invalidEnd:
		r.invalidTraffic(base, iter)
	case roll < ddosEnd:
		r.ddosTraffic(base, iter)
	default:
		r.edgeTraffic(base, iter)
	}
}

func openrtbBidBody(iter uint64) []byte {
	id := iter % 100000
	return []byte(`{"id":"load-` + strconv.FormatUint(id, 10) + `","tmax":300,"imp":[{"id":"1","bidfloor":0.5,"banner":{"w":300,"h":250}}],"site":{"page":"https://example.com"},"device":{"ip":"8.8.8.8","ua":"Mozilla/5.0","devicetype":2,"geo":{"country":"US"}}}`)
}

func (r *runner) telegramTraffic(base string, iter uint64) {
	cid := r.pickCampaign(iter)
	clickID := fmt.Sprintf("00000000-0000-4000-8000-%012x", iter)
	token := "token_abc123_"
	switch iter % 5 {
	case 0:
		r.get(fmt.Sprintf("%s/tg/click?campaign_id=%s&click_id=%s&bridge_token=%s", base, cid, clickID, token))
	case 1:
		r.get(fmt.Sprintf("%s/tg/impression?campaign_id=%s&click_id=%s", base, cid, clickID))
	case 2:
		r.get(fmt.Sprintf("%s/tg/click?campaign_id=%s&click_id=%s&initData=evil", base, cid, clickID))
	case 3:
		body := []byte(`{"ip":"8.8.8.8","user_agent":"TelegramBot/1.0","publisher_id":"pub1","bid_floor":0.1,"premium":true}`)
		r.post(base+"/tg/bid", "application/json", body, nil)
	default:
		r.get(fmt.Sprintf("%s/tg/click?campaign_id=%s&click_id=not-a-uuid&bridge_token=%s", base, cid, token))
	}
}

// clickProxyTraffic issues GET /click. Exercises proxy delivery only when the
// pinned campaign is configured with click_delivery=proxy and a mock upstream;
// otherwise it measures the redirect path.
func (r *runner) clickProxyTraffic(base string, iter uint64) {
	cid := r.pickCampaign(iter)
	clickID := fmt.Sprintf("00000000-0000-4000-8000-%012x", iter)
	r.get(fmt.Sprintf("%s/click?campaign_id=%s&click_id=%s&sub1=loadgen", base, cid, clickID))
}

func (r *runner) postOpenRTBBid(base string, iter uint64) {
	body := openrtbBidBody(iter)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, base+"/openrtb/bid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-openrtb-version", "2.6")
	req.Header.Set("Connection", "keep-alive")
	r.exec(req)
}

func (r *runner) validBody(iter uint64) []byte {
	b, _ := json.Marshal(map[string]any{
		"campaign_id": r.pickCampaign(iter),
		"user_id":     fmt.Sprintf("u-%d", iter),
		"type":        ternary(iter%3 == 0, "click", "impression"),
		"click_id":    fmt.Sprintf("clk-%d", iter),
		"payload":     map[string]any{"slot": "top", "cpm": 1.25},
	})
	return b
}

func (r *runner) fraudBody(iter uint64) []byte {
	b, _ := json.Marshal(map[string]any{
		"campaign_id": r.pickCampaign(iter),
		"user_id":     fmt.Sprintf("fraud-%d", iter),
		"type":        "click",
		"click_id":    fmt.Sprintf("fclk-%d", iter),
		"payload":     map[string]any{"bot": true},
	})
	return b
}

func fraudIP(iter uint64) string {
	return fmt.Sprintf("54.%d.%d.%d", (iter>>8)&255, iter&255, (iter>>16)&255)
}

func (r *runner) invalidTraffic(base string, iter uint64) {
	switch iter % 4 {
	case 0:
		r.post(base+"/track", "application/json", []byte("{not-json"), nil)
	case 1:
		r.post(base+"/track", "application/x-protobuf", []byte{0xff, 0xee, 0xdd, 0xcc, 0xbb}, nil)
	case 2:
		b, _ := json.Marshal(map[string]any{
			"campaign_id": "ffffffff-ffff-ffff-ffff-ffffffffffff",
			"user_id":     "ghost",
			"type":        "impression",
			"click_id":    fmt.Sprintf("bad-%d", iter),
		})
		r.post(base+"/track", "application/json", b, nil)
	default:
		big := strings.Repeat("x", r.oversizeBytes)
		r.post(base+"/track", "application/json", []byte(big), nil)
	}
}

func (r *runner) ddosTraffic(base string, iter uint64) {
	switch iter % 5 {
	case 0:
		r.get(base + "/track")
	case 1:
		r.get(base + "/health")
	case 2:
		r.get(base + "/metrics")
	case 3:
		dup, _ := json.Marshal(map[string]any{
			"campaign_id": campaignID(1),
			"user_id":     "ddos-dup",
			"type":        "click",
			"click_id":    "dup-fixed-id",
			"payload":     map[string]any{},
		})
		r.post(base+"/track", "application/json", dup, nil)
	default:
		r.post(base+"/admin/boom", "application/json", []byte("{}"), nil)
	}
}

func (r *runner) edgeTraffic(base string, iter uint64) {
	if r.edgeURL != "" {
		r.post(r.edgeURL+"/track", "application/json", r.validBody(iter), nil)
		return
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, base+"/track", http.NoBody)
	req.Header.Set("Content-Type", "application/json")
	r.exec(req)
}

func (r *runner) get(url string) {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
	r.exec(req)
}

func (r *runner) post(url, ctype string, body []byte, extra map[string]string) {
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("Connection", "keep-alive")
	for k, v := range extra {
		req.Header.Set(k, v)
	}
	r.exec(req)
}

func (r *runner) exec(req *http.Request) {
	resp, err := r.client.Do(req)
	if err != nil {
		r.hist.inc("0", classifyNetErr(err))
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	r.hist.inc(strconv.Itoa(resp.StatusCode), "none")
}

func classifyNetErr(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "connection reset"):
		return "conn_reset"
	case strings.Contains(s, "broken pipe"):
		return "broken_pipe"
	case strings.Contains(s, "timeout"), strings.Contains(s, "deadline"):
		return "timeout"
	case strings.Contains(s, "EOF"):
		return "eof"
	case strings.Contains(s, "connection refused"), strings.Contains(s, "dial"):
		return "dial"
	default:
		return "other"
	}
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func healthCheck(trackers []string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, base := range trackers {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, base+"/health", http.NoBody)
		if err != nil {
			return fmt.Errorf("%s: %w", base, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", base, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s: status %d", base, resp.StatusCode)
		}
	}
	return nil
}
