package integrationschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/postback"
	"gopkg.in/yaml.v3"
)

const (
	MaxBodyBytes   = 64 * 1024
	MaxURLTemplate = postback.MaxRenderedURLLen
	MaxTokenLen    = 256
	MaxNameLen     = 128
)

type Kind string

const (
	KindInboundTokens           Kind = "inbound_tokens"
	KindOutboundPostback        Kind = "outbound_postback"
	KindAffiliateReceivePostback Kind = "affiliate_receive_postback"
	KindStatusMapping           Kind = "status_mapping"
)

type InboundTokenDef struct {
	Name     string `json:"name"`
	QueryKey string `json:"query_key"`
	MaxLen   int    `json:"max_len"`
}

type InboundMacroDef struct {
	Name     string `json:"name"`
	Source   string `json:"source"`
	Key      string `json:"key"`
	Required bool   `json:"required"`
}

type InboundTokensSchema struct {
	Version int               `json:"version"`
	Tokens  []InboundTokenDef `json:"tokens"`
	Macros  []InboundMacroDef `json:"macros"`
}

type OutboundPostbackSchema struct {
	Version      int      `json:"version"`
	URLTemplate  string   `json:"url_template"`
	Placeholders []string `json:"placeholders"`
}

type AffiliateReceivePostbackSchema struct {
	Version            int    `json:"version"`
	ReceiveURLTemplate string `json:"receive_url_template"`
	OfferURLSuffix     string `json:"offer_url_suffix,omitempty"`
}

type StatusMappingSchema struct {
	Version   int               `json:"version"`
	StatusMap map[string]string `json:"status_map"`
}

var outboundPlaceholderAllowlist = map[string]struct{}{
	"click_id":   {},
	"payout":     {},
	"tx_id":      {},
	"currency":   {},
	"status":     {},
	"event_type": {},
	"param10":    {},
}

func init() {
	for i := 1; i <= 30; i++ {
		outboundPlaceholderAllowlist[fmt.Sprintf("sub%d", i)] = struct{}{}
		outboundPlaceholderAllowlist[fmt.Sprintf("subid%d", i)] = struct{}{}
	}
}

func ParseDocument(raw []byte) (Kind, any, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", nil, errors.New("empty schema body")
	}
	if len(raw) > MaxBodyBytes {
		return "", nil, fmt.Errorf("schema body exceeds %d bytes", MaxBodyBytes)
	}

	jsonBytes := raw
	if raw[0] != '{' && raw[0] != '[' {
		var decoded any
		if err := yaml.Unmarshal(raw, &decoded); err != nil {
			return "", nil, fmt.Errorf("invalid yaml: %w", err)
		}
		var err error
		jsonBytes, err = json.Marshal(decoded)
		if err != nil {
			return "", nil, err
		}
	}

	kind, err := detectKind(jsonBytes)
	if err != nil {
		return "", nil, err
	}

	switch kind {
	case KindInboundTokens:
		var s InboundTokensSchema
		if err := decodeStrict(jsonBytes, &s); err != nil {
			return "", nil, err
		}
		if err := validateInboundTokens(&s); err != nil {
			return "", nil, err
		}
		return kind, &s, nil
	case KindOutboundPostback:
		var s OutboundPostbackSchema
		if err := decodeStrict(jsonBytes, &s); err != nil {
			return "", nil, err
		}
		if err := validateOutboundPostback(&s); err != nil {
			return "", nil, err
		}
		return kind, &s, nil
	case KindAffiliateReceivePostback:
		var s AffiliateReceivePostbackSchema
		if err := decodeStrict(jsonBytes, &s); err != nil {
			return "", nil, err
		}
		if err := validateAffiliateReceivePostback(&s); err != nil {
			return "", nil, err
		}
		return kind, &s, nil
	case KindStatusMapping:
		var s StatusMappingSchema
		if err := decodeStrict(jsonBytes, &s); err != nil {
			return "", nil, err
		}
		if err := validateStatusMapping(&s); err != nil {
			return "", nil, err
		}
		return kind, &s, nil
	default:
		return "", nil, fmt.Errorf("unsupported schema kind %q", kind)
	}
}

func detectKind(jsonBytes []byte) (Kind, error) {
	var probe struct {
		Tokens             []json.RawMessage `json:"tokens"`
		Placeholders       []string          `json:"placeholders"`
		URLTemplate        string            `json:"url_template"`
		ReceiveURLTemplate string            `json:"receive_url_template"`
		StatusMap          map[string]string `json:"status_map"`
	}
	if err := json.Unmarshal(jsonBytes, &probe); err != nil {
		return "", fmt.Errorf("invalid schema json: %w", err)
	}
	switch {
	case len(probe.Tokens) > 0:
		return KindInboundTokens, nil
	case probe.ReceiveURLTemplate != "":
		return KindAffiliateReceivePostback, nil
	case probe.URLTemplate != "" || len(probe.Placeholders) > 0:
		return KindOutboundPostback, nil
	case len(probe.StatusMap) > 0:
		return KindStatusMapping, nil
	default:
		return "", errors.New("cannot detect schema kind (expected tokens, receive_url_template, url_template/placeholders, or status_map)")
	}
}

func decodeStrict(jsonBytes []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid schema fields: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return errors.New("schema must be a single JSON object")
	}
	return nil
}

func validateInboundTokens(s *InboundTokensSchema) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported inbound schema version %d", s.Version)
	}
	if len(s.Tokens) == 0 {
		return errors.New("inbound schema requires at least one token")
	}
	seen := make(map[string]struct{}, len(s.Tokens))
	for _, t := range s.Tokens {
		name := strings.TrimSpace(t.Name)
		key := strings.TrimSpace(t.QueryKey)
		if name == "" || key == "" {
			return errors.New("token name and query_key are required")
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("duplicate token name %q", name)
		}
		seen[name] = struct{}{}
		maxLen := t.MaxLen
		if maxLen <= 0 {
			maxLen = MaxTokenLen
		}
		if maxLen > MaxTokenLen {
			return fmt.Errorf("token %q max_len exceeds %d", name, MaxTokenLen)
		}
	}
	for _, m := range s.Macros {
		if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Key) == "" {
			return errors.New("macro name and key are required")
		}
		if src := strings.TrimSpace(m.Source); src != "" && src != "query" {
			return fmt.Errorf("unsupported macro source %q", src)
		}
	}
	return nil
}

func validateAffiliateReceivePostback(s *AffiliateReceivePostbackSchema) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported affiliate receive schema version %d", s.Version)
	}
	tpl := strings.TrimSpace(s.ReceiveURLTemplate)
	if tpl == "" {
		return errors.New("affiliate receive schema requires receive_url_template")
	}
	if len(tpl) > MaxURLTemplate {
		return fmt.Errorf("receive_url_template exceeds %d bytes", MaxURLTemplate)
	}
	if !strings.Contains(tpl, "{tracking_domain}") {
		return errors.New("receive_url_template must include {tracking_domain}")
	}
	return nil
}

func BuildAffiliateReceivePanelURL(trackingDomain string, s *AffiliateReceivePostbackSchema) string {
	host := strings.TrimSpace(trackingDomain)
	if host == "" {
		host = "track.example.com"
	}
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	return strings.ReplaceAll(s.ReceiveURLTemplate, "{tracking_domain}", host)
}

func validateOutboundPostback(s *OutboundPostbackSchema) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported outbound schema version %d", s.Version)
	}
	tpl := strings.TrimSpace(s.URLTemplate)
	if tpl == "" {
		return errors.New("outbound schema requires url_template")
	}
	if len(tpl) > MaxURLTemplate {
		return fmt.Errorf("url_template exceeds %d bytes", MaxURLTemplate)
	}
	if len(s.Placeholders) == 0 {
		return errors.New("outbound schema requires placeholders")
	}
	allowed := make(map[string]struct{}, len(s.Placeholders))
	for _, ph := range s.Placeholders {
		ph = strings.TrimSpace(ph)
		if ph == "" {
			return errors.New("empty placeholder name")
		}
		if _, ok := outboundPlaceholderAllowlist[ph]; !ok {
			return fmt.Errorf("unknown placeholder %q", ph)
		}
		allowed[ph] = struct{}{}
	}
	for _, used := range extractTemplatePlaceholders(tpl) {
		if _, ok := allowed[used]; !ok {
			return fmt.Errorf("url_template uses undeclared placeholder %q", used)
		}
	}
	_ = postback.ParseTemplate(tpl)
	return nil
}

func validateStatusMapping(s *StatusMappingSchema) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported status schema version %d", s.Version)
	}
	if len(s.StatusMap) == 0 {
		return errors.New("status_map is required")
	}
	for ext, mapped := range s.StatusMap {
		if strings.TrimSpace(ext) == "" || strings.TrimSpace(mapped) == "" {
			return errors.New("status_map keys and values must be non-empty")
		}
	}
	return nil
}

func extractTemplatePlaceholders(tpl string) []string {
	var out []string
	for i := 0; i < len(tpl); {
		start := strings.IndexByte(tpl[i:], '{')
		if start < 0 {
			break
		}
		start += i
		end := strings.IndexByte(tpl[start+1:], '}')
		if end < 0 {
			break
		}
		end += start + 1
		out = append(out, tpl[start+1:end])
		i = end + 1
	}
	return out
}

func MapAffiliateStatus(s *StatusMappingSchema, external string) (string, bool) {
	if s == nil || len(s.StatusMap) == 0 {
		return "", false
	}
	mapped, ok := s.StatusMap[strings.ToLower(strings.TrimSpace(external))]
	return mapped, ok
}

func BuildInboundClickQuery(s *InboundTokensSchema, values map[string]string) (string, error) {
	if s == nil {
		return "", errors.New("nil inbound schema")
	}
	var parts []string
	for _, m := range s.Macros {
		val := strings.TrimSpace(values[m.Key])
		if val == "" && m.Required {
			return "", fmt.Errorf("required macro %q missing", m.Name)
		}
		if val != "" {
			parts = append(parts, m.Key+"="+val)
		}
	}
	for _, t := range s.Tokens {
		val := strings.TrimSpace(values[t.QueryKey])
		if val == "" {
			continue
		}
		maxLen := t.MaxLen
		if maxLen <= 0 {
			maxLen = MaxTokenLen
		}
		if len(val) > maxLen {
			return "", fmt.Errorf("token %q exceeds max_len %d", t.Name, maxLen)
		}
		parts = append(parts, t.QueryKey+"="+val)
	}
	return strings.Join(parts, "&"), nil
}

func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > MaxNameLen {
		return fmt.Errorf("name exceeds %d characters", MaxNameLen)
	}
	return nil
}

func OutboundURLTemplateFromBody(body json.RawMessage) (string, error) {
	kind, parsed, err := ParseDocument(body)
	if err != nil {
		return "", err
	}
	if kind != KindOutboundPostback {
		return "", fmt.Errorf("schema kind %q is not outbound_postback", kind)
	}
	s := parsed.(*OutboundPostbackSchema)
	return strings.TrimSpace(s.URLTemplate), nil
}
