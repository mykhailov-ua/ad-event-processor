package track

import (
	"strings"

	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

type DecoyTemplateSource uint8

const (
	DecoySourceStatic DecoyTemplateSource = 0
	DecoySourceHosted DecoyTemplateSource = 1
	DecoySourceURL    DecoyTemplateSource = 2
)

type DecoyTemplateInput struct {
	DecoyLanderID uuid.UUID
	SafePageURL   string
}

var SafeDecoyStaticBody = []byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>Loading</title></head><body><main id=\"aed-mount\"><p>Loading</p></main><script src=\"/static/tag-ev.js\"></script><script src=\"/static/tag-ctx.js\"></script></body></html>")

func (s DecoyTemplateSource) MetricLabel() string {
	switch s {
	case DecoySourceHosted:
		return "hosted"
	case DecoySourceURL:
		return "url"
	default:
		return "static"
	}
}

func HostedLanderPath(landerID uuid.UUID) string {
	return "/lp/" + landerID.String() + "/"
}

func ParseHostedLanderIDFromURL(url string) uuid.UUID {
	url = strings.TrimSpace(url)
	if url == "" {
		return uuid.Nil
	}
	const marker = "/lp/"
	idx := strings.Index(url, marker)
	if idx < 0 {
		return uuid.Nil
	}
	start := idx + len(marker)
	if start+36 > len(url) {
		return uuid.Nil
	}
	id, err := uuid.Parse(url[start : start+36])
	if err != nil {
		return uuid.Nil
	}
	return id
}

func ResolveDecoyTemplate(in DecoyTemplateInput) (DecoyTemplateSource, []byte) {
	if in.DecoyLanderID != uuid.Nil {
		return DecoySourceHosted, []byte(HostedLanderPath(in.DecoyLanderID))
	}
	if derived := ParseHostedLanderIDFromURL(in.SafePageURL); derived != uuid.Nil {
		return DecoySourceHosted, []byte(HostedLanderPath(derived))
	}
	if in.SafePageURL != "" {
		if urlBytes, ok := safePageURLAttrBytes(in.SafePageURL); ok {
			return DecoySourceURL, urlBytes
		}
	}
	return DecoySourceStatic, nil
}

func AppendDecoyBody(dst []byte, source DecoyTemplateSource, iframeURL []byte) []byte {
	if source == DecoySourceStatic || len(iframeURL) == 0 {
		return append(dst, SafeDecoyStaticBody...)
	}
	return AppendSafePageDecoyBody(dst, iframeURL)
}

func BuildDecoyBody(in DecoyTemplateInput) []byte {
	source, url := ResolveDecoyTemplate(in)
	metrics.SafePageDecoyTemplateTotal.WithLabelValues(source.MetricLabel()).Inc()
	return AppendDecoyBody(nil, source, url)
}

func BuildDecoyBodyNoMetric(in DecoyTemplateInput) []byte {
	source, url := ResolveDecoyTemplate(in)
	return AppendDecoyBody(nil, source, url)
}
