package track

import (
	"encoding/json"
	"strconv"

	"ad-event-processor/internal/domain"
)

type analyticsDimensions struct {
	sub1       string
	sub2       string
	country    string
	deviceType string
	keyword    string
	payload    []byte
}

type AnalyticsDimensions struct {
	Sub1       string
	Sub2       string
	Country    string
	DeviceType string
	Keyword    string
	Payload    []byte
}

func ExtractAnalyticsDimensions(evt *domain.Event) AnalyticsDimensions {
	d := extractAnalyticsDimensions(evt)
	return AnalyticsDimensions{
		Sub1:       d.sub1,
		Sub2:       d.sub2,
		Country:    d.country,
		DeviceType: d.deviceType,
		Keyword:    d.keyword,
		Payload:    d.payload,
	}
}

func AnalyticsCountryCode(country string) string {
	return analyticsCountryCode(country)
}

func AnalyticsPayloadBytes(dims AnalyticsDimensions, fallback []byte) []byte {
	return analyticsPayloadBytes(analyticsDimensions{
		sub1:       dims.Sub1,
		sub2:       dims.Sub2,
		country:    dims.Country,
		deviceType: dims.DeviceType,
		keyword:    dims.Keyword,
		payload:    dims.Payload,
	}, fallback)
}

func enrichAnalyticsPayload(evt *domain.Event) []byte {
	return extractAnalyticsDimensions(evt).payload
}

func extractAnalyticsDimensions(evt *domain.Event) analyticsDimensions {
	if evt == nil {
		return analyticsDimensions{}
	}
	fields := parsePayloadStringFields(evt.Payload)
	dims := analyticsDimensions{
		sub1:       fields["sub1"],
		sub2:       fields["sub2"],
		country:    fields["country"],
		deviceType: firstNonEmpty(fields["device_type"], fields["device"]),
		keyword:    fields["keyword"],
	}
	if dims.country == "" && evt.GeoCountry != "" {
		dims.country = evt.GeoCountry
	}
	if fields["geo_hash"] == "" && evt.GeoHash != 0 {
		fields["geo_hash"] = strconv.FormatUint(uint64(evt.GeoHash), 10)
	}
	if dims.country != "" && fields["country"] == "" {
		fields["country"] = dims.country
	}
	if dims.deviceType != "" && fields["device_type"] == "" {
		fields["device_type"] = dims.deviceType
	}
	dims.payload = mergePayloadFields(evt.Payload, fields)
	return dims
}

func ParsePayloadStringFields(payload []byte) map[string]string {
	return parsePayloadStringFields(payload)
}

func parsePayloadStringFields(payload []byte) map[string]string {
	out := make(map[string]string)
	if len(payload) == 0 {
		return out
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return out
	}
	for key, val := range raw {
		if len(val) == 0 || string(val) == "null" {
			continue
		}
		var s string
		if err := json.Unmarshal(val, &s); err == nil && s != "" {
			out[key] = s
		}
	}
	return out
}

func mergePayloadFields(original []byte, fields map[string]string) []byte {
	if len(fields) == 0 {
		if len(original) == 0 {
			return nil
		}
		return append([]byte(nil), original...)
	}
	merged := make(map[string]string, len(fields))
	for k, v := range parsePayloadStringFields(original) {
		merged[k] = v
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if existing, ok := merged[k]; ok && existing != "" {
			continue
		}
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	out, err := json.Marshal(merged)
	if err != nil {
		if len(original) > 0 {
			return append([]byte(nil), original...)
		}
		return nil
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func analyticsCountryCode(country string) string {
	if country == "" {
		return ""
	}
	if len(country) >= 2 {
		b0, b1 := country[0], country[1]
		if b0 >= 'a' && b0 <= 'z' {
			b0 -= 'a' - 'A'
		}
		if b1 >= 'a' && b1 <= 'z' {
			b1 -= 'a' - 'A'
		}
		return string([]byte{b0, b1})
	}
	return country
}

func analyticsPayloadBytes(dims analyticsDimensions, fallback []byte) []byte {
	if len(dims.payload) > 0 {
		return dims.payload
	}
	if len(fallback) > 0 {
		return fallback
	}
	return nil
}
