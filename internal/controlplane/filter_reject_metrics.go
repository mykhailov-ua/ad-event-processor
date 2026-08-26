package controlplane

import (
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	filterBlockedMetricName       = "ad_filter_blocked_total"
	filterRejectCountryMetricName = "ad_filter_reject_country_total"
)

type filterRejectCounterSample struct {
	Kind  string
	Value float64
}

type filterRejectSliceSample struct {
	Kind    string
	Country string
	Value   float64
}

type filterRejectMetricsSnapshot struct {
	Totals map[string]float64
	Slices []filterRejectSliceSample
}

func parseFilterRejectCounters(r io.Reader, contentType string) ([]filterRejectCounterSample, error) {
	snap, err := parseFilterRejectMetrics(r, contentType)
	if err != nil {
		return nil, err
	}
	out := make([]filterRejectCounterSample, 0, len(snap.Totals))
	for kind, val := range snap.Totals {
		out = append(out, filterRejectCounterSample{Kind: kind, Value: val})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out, nil
}

func parseFilterRejectMetrics(r io.Reader, contentType string) (filterRejectMetricsSnapshot, error) {
	format := expfmt.NewFormat(expfmt.TypeTextPlain)
	if contentType != "" {
		if parsed := expfmt.ResponseFormat(http.Header{"Content-Type": {contentType}}); parsed != expfmt.FmtUnknown {
			format = parsed
		}
	}
	dec := expfmt.NewDecoder(r, format)
	totals := make(map[string]float64)
	var slices []filterRejectSliceSample
	for {
		var mf dto.MetricFamily
		if err := dec.Decode(&mf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return filterRejectMetricsSnapshot{}, err
		}
		if mf.Name == nil {
			continue
		}
		switch *mf.Name {
		case filterBlockedMetricName:
			for _, m := range mf.Metric {
				kind := prometheusLabelValue(m.Label, "reason")
				if kind == "" {
					continue
				}
				val, ok := prometheusMetricValue(m)
				if !ok {
					continue
				}
				totals[kind] += val
			}
		case filterRejectCountryMetricName:
			for _, m := range mf.Metric {
				kind := prometheusLabelValue(m.Label, "reason")
				country := prometheusLabelValue(m.Label, "country")
				if kind == "" || country == "" {
					continue
				}
				val, ok := prometheusMetricValue(m)
				if !ok {
					continue
				}
				slices = append(slices, filterRejectSliceSample{Kind: kind, Country: country, Value: val})
			}
		}
	}
	sort.Slice(slices, func(i, j int) bool {
		if slices[i].Kind != slices[j].Kind {
			return slices[i].Kind < slices[j].Kind
		}
		return slices[i].Country < slices[j].Country
	})
	return filterRejectMetricsSnapshot{Totals: totals, Slices: slices}, nil
}

func prometheusLabelValue(labels []*dto.LabelPair, name string) string {
	for _, lp := range labels {
		if lp == nil || lp.Name == nil || lp.Value == nil {
			continue
		}
		if *lp.Name == name {
			return *lp.Value
		}
	}
	return ""
}

func prometheusMetricValue(m *dto.Metric) (float64, bool) {
	if m == nil {
		return 0, false
	}
	switch {
	case m.Counter != nil:
		return m.Counter.GetValue(), true
	case m.Gauge != nil:
		return m.Gauge.GetValue(), true
	case m.Untyped != nil:
		return m.Untyped.GetValue(), true
	default:
		return 0, false
	}
}

func filterRejectCounterDelta(previous, current float64) uint64 {
	if current <= previous {
		if current < previous {
			return uint64(current)
		}
		return 0
	}
	return uint64(current - previous)
}

func mergeFilterRejectCounterSamples(samples []filterRejectCounterSample) map[string]float64 {
	out := make(map[string]float64, len(samples))
	for _, sample := range samples {
		kind := strings.TrimSpace(sample.Kind)
		if kind == "" {
			continue
		}
		out[kind] += sample.Value
	}
	return out
}

func filterRejectSliceKey(kind, country string) string {
	return kind + "|" + country
}

func mergeFilterRejectSliceSamples(samples []filterRejectSliceSample) map[string]float64 {
	out := make(map[string]float64, len(samples))
	for _, sample := range samples {
		kind := strings.TrimSpace(sample.Kind)
		country := strings.TrimSpace(sample.Country)
		if kind == "" || country == "" {
			continue
		}
		out[filterRejectSliceKey(kind, country)] += sample.Value
	}
	return out
}

func splitFilterRejectSliceKey(key string) (kind, country string) {
	kind, country, ok := strings.Cut(key, "|")
	if !ok {
		return key, ""
	}
	return kind, country
}
