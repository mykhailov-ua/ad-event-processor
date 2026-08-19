package integrationschema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"
)

type TemplateCatalogEntry struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	Version  int    `json:"version"`
	Category string `json:"category"`
	Kind     Kind   `json:"kind"`
}

var GMM4TemplateCatalog = []TemplateCatalogEntry{
	{Name: "traffic_propellerads", File: "traffic_propellerads.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_exoclick", File: "traffic_exoclick.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "traffic_facebook", File: "traffic_facebook.v1.yaml", Version: 1, Category: "traffic_source", Kind: KindInboundTokens},
	{Name: "affiliate_everad", File: "affiliate_everad_postback.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindOutboundPostback},
	{Name: "affiliate_everad_status", File: "affiliate_everad_status.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindStatusMapping},
	{Name: "affiliate_leadbit", File: "affiliate_leadbit_postback.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindOutboundPostback},
	{Name: "affiliate_leadbit_status", File: "affiliate_leadbit_status.v1.yaml", Version: 1, Category: "affiliate_network", Kind: KindStatusMapping},
}

var affiliateStatusPairs = map[string]string{
	"affiliate_everad":  "affiliate_everad_status",
	"affiliate_leadbit": "affiliate_leadbit_status",
}

func AffiliateStatusTemplateName(network string) (string, bool) {
	name, ok := affiliateStatusPairs[strings.TrimSpace(network)]
	return name, ok
}

func SchemaRootDir() string {
	if root := config.InstallRootFromEnv(); root != "" {
		return filepath.Join(root, "deploy", "schemas")
	}
	candidates := []string{
		filepath.Join("deploy", "schemas"),
		filepath.Join("..", "..", "deploy", "schemas"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

func LoadBundledTemplate(entry TemplateCatalogEntry) (raw []byte, kind Kind, schema any, err error) {
	raw, err = os.ReadFile(filepath.Join(SchemaRootDir(), entry.File))
	if err != nil {
		return nil, "", nil, fmt.Errorf("read %s: %w", entry.File, err)
	}
	kind, parsed, err := ParseDocument(raw)
	if err != nil {
		return nil, "", nil, err
	}
	if entry.Kind != "" && kind != entry.Kind {
		return nil, "", nil, fmt.Errorf("template %s: expected kind %s, got %s", entry.Name, entry.Kind, kind)
	}
	return raw, kind, parsed, nil
}

func FindCatalogEntry(name string) (TemplateCatalogEntry, bool) {
	name = strings.TrimSpace(name)
	for _, e := range GMM4TemplateCatalog {
		if e.Name == name {
			return e, true
		}
	}
	return TemplateCatalogEntry{}, false
}

func BuildInboundTrackingURL(trackingDomain string, s *InboundTokensSchema) string {
	host := platformconfig.ResolveHost(trackingDomain)
	if host == "" {
		host = "track.example.com"
	}
	var parts []string
	parts = append(parts, "campaign_id={campaign_id}")
	seen := map[string]struct{}{"campaign_id": {}}
	for _, t := range s.Tokens {
		key := strings.TrimSpace(t.QueryKey)
		if key == "" || key == "campaign_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"={"+key+"}")
	}
	for _, m := range s.Macros {
		key := strings.TrimSpace(m.Key)
		if key == "" || key == "campaign_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"={"+key+"}")
	}
	return fmt.Sprintf("https://%s/click?%s", host, strings.Join(parts, "&"))
}
