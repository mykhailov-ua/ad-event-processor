package controlplane

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type scrapedMetric struct {
	Name       string
	LabelsHash string
	Value      float64
}

var opsScrapeMetricNames = map[string]struct{}{
	"ad_http_requests_total":             {},
	"ad_recon_drift_micro":               {},
	"ad_management_outbox_pending_total": {},
	"ad_tracker_redis_shard_healthy":     {},
}

func parsePrometheusMetrics(r io.Reader, contentType string) ([]scrapedMetric, error) {
	format := expfmt.NewFormat(expfmt.TypeTextPlain)
	if contentType != "" {
		if parsed := expfmt.ResponseFormat(http.Header{"Content-Type": {contentType}}); parsed != expfmt.FmtUnknown {
			format = parsed
		}
	}
	dec := expfmt.NewDecoder(r, format)
	var out []scrapedMetric
	var maxDrift float64
	var sawDrift bool
	for {
		var mf dto.MetricFamily
		if err := dec.Decode(&mf); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if mf.Name == nil {
			continue
		}
		name := *mf.Name
		if _, ok := opsScrapeMetricNames[name]; !ok {
			continue
		}
		for _, m := range mf.Metric {
			val, ok := metricValue(m)
			if !ok {
				continue
			}
			if name == "ad_recon_drift_micro" {
				sawDrift = true
				if val > maxDrift {
					maxDrift = val
				}
				continue
			}
			out = append(out, scrapedMetric{
				Name:       name,
				LabelsHash: labelsHash(m.Label),
				Value:      val,
			})
		}
	}
	if sawDrift {
		out = append(out, scrapedMetric{
			Name:       "ad_recon_drift_micro_max",
			LabelsHash: "",
			Value:      maxDrift,
		})
	}
	return out, nil
}

func metricValue(m *dto.Metric) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch {
	case m.Gauge != nil:
		return m.Gauge.GetValue(), true
	case m.Counter != nil:
		return m.Counter.GetValue(), true
	case m.Untyped != nil:
		return m.Untyped.GetValue(), true
	default:
		return 0, false
	}
}

func labelsHash(labels []*dto.LabelPair) string {
	if len(labels) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(labels))
	for _, lp := range labels {
		if lp == nil || lp.Name == nil || lp.Value == nil {
			continue
		}
		pairs = append(pairs, *lp.Name+"="+*lp.Value)
	}
	if len(pairs) == 0 {
		return ""
	}
	sort.Strings(pairs)
	sum := sha256.Sum256([]byte(strings.Join(pairs, "|")))
	return hex.EncodeToString(sum[:8])
}
