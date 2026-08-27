package traffictemplates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ad-event-processor/internal/integrationschema"

	"gopkg.in/yaml.v3"
)

const directCustomID = "direct-custom"

func Generate(schemasDir string, sidecarPath string) ([]Template, error) {
	sidecar, err := LoadSidecar(sidecarPath)
	if err != nil {
		return nil, err
	}
	coveredSlugs := make(map[string]struct{}, len(sidecar.Templates))
	for _, tpl := range sidecar.Templates {
		if slug := strings.TrimSpace(tpl.BundledSlug); slug != "" {
			coveredSlugs[slug] = struct{}{}
		}
	}

	auto := make([]Template, 0, 64)
	for _, entry := range integrationschema.BundledIntegrationTemplateCatalog {
		if entry.Category != "traffic_source" {
			continue
		}
		if _, ok := coveredSlugs[entry.Name]; ok {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(schemasDir, entry.File))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.File, err)
		}
		_, parsed, err := integrationschema.ParseDocument(raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.File, err)
		}
		inbound, ok := parsed.(*integrationschema.InboundTokensSchema)
		if !ok {
			return nil, fmt.Errorf("template %s: expected inbound_tokens", entry.Name)
		}
		auto = append(auto, templateFromSchema(entry.Name, inbound))
	}

	out := make([]Template, 0, len(sidecar.Templates)+len(auto)+1)
	out = append(out, sidecar.Templates...)
	out = append(out, auto...)
	out = append(out, Template{
		ID:       directCustomID,
		Name:     "Direct / Custom",
		Category: "direct",
		Notes:    "Manual sub_ids - fill values or leave empty.",
		Params:   nil,
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == directCustomID {
			return false
		}
		if out[j].ID == directCustomID {
			return true
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func LoadSidecar(path string) (*SidecarFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sidecar %s: %w", path, err)
	}
	var doc SidecarFile
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("decode sidecar %s: %w", path, err)
	}
	if doc.Version <= 0 {
		return nil, fmt.Errorf("sidecar %s: version required", path)
	}
	for i := range doc.Templates {
		if strings.TrimSpace(doc.Templates[i].ID) == "" {
			return nil, fmt.Errorf("sidecar template index %d missing id", i)
		}
		if strings.TrimSpace(doc.Templates[i].Name) == "" {
			return nil, fmt.Errorf("sidecar template %s missing name", doc.Templates[i].ID)
		}
		if strings.TrimSpace(doc.Templates[i].Category) == "" {
			return nil, fmt.Errorf("sidecar template %s missing category", doc.Templates[i].ID)
		}
	}
	return &doc, nil
}

func templateFromSchema(bundledSlug string, schema *integrationschema.InboundTokensSchema) Template {
	id := defaultTemplateID(bundledSlug)
	params := make([]Param, 0, len(schema.Tokens)+len(schema.Macros))
	seen := map[string]struct{}{"campaign_id": {}}
	for _, token := range schema.Tokens {
		key := strings.TrimSpace(token.QueryKey)
		if key == "" || key == "campaign_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		name := strings.TrimSpace(token.Name)
		if name == "" {
			name = key
		}
		params = append(params, Param{
			Key:   key,
			Value: "{" + name + "}",
			Label: humanLabel(name),
		})
	}
	for _, macro := range schema.Macros {
		key := strings.TrimSpace(macro.Key)
		if key == "" || key == "campaign_id" || key == "click_id" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		params = append(params, Param{Key: key, Value: "{" + key + "}"})
	}
	sort.Slice(params, func(i, j int) bool { return params[i].Key < params[j].Key })
	return Template{
		ID:          id,
		BundledSlug: bundledSlug,
		Name:        defaultTemplateName(bundledSlug),
		Category:    defaultCategory(bundledSlug),
		Params:      params,
		Generated:   true,
	}
}

func defaultTemplateID(bundledSlug string) string {
	slug := strings.TrimPrefix(strings.TrimSpace(bundledSlug), "traffic_")
	return strings.ReplaceAll(slug, "_", "-")
}

func defaultTemplateName(bundledSlug string) string {
	slug := strings.TrimPrefix(strings.TrimSpace(bundledSlug), "traffic_")
	parts := strings.Split(slug, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func humanLabel(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	if name == "" {
		return ""
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func defaultCategory(bundledSlug string) string {
	s := strings.ToLower(bundledSlug)
	switch {
	case strings.Contains(s, "facebook"), strings.Contains(s, "instagram"), strings.Contains(s, "snapchat"),
		strings.Contains(s, "tiktok"), strings.Contains(s, "youtube"), strings.Contains(s, "linkedin"),
		strings.Contains(s, "pinterest"), strings.Contains(s, "x_ads"), strings.Contains(s, "threads"),
		strings.Contains(s, "reddit"):
		return "social"
	case strings.Contains(s, "google"), strings.Contains(s, "microsoft"), strings.Contains(s, "yandex"):
		return "search"
	case strings.Contains(s, "taboola"), strings.Contains(s, "outbrain"), strings.Contains(s, "mgid"),
		strings.Contains(s, "revcontent"), strings.Contains(s, "mediago"), strings.Contains(s, "adskeeper"):
		return "native"
	case strings.Contains(s, "exoclick"), strings.Contains(s, "trafficjunky"), strings.Contains(s, "juicyads"),
		strings.Contains(s, "trafficstars"), strings.Contains(s, "plugrush"):
		return "adult"
	case strings.Contains(s, "propeller"), strings.Contains(s, "push"), strings.Contains(s, "pop"),
		strings.Contains(s, "clickadu"), strings.Contains(s, "zeropark"), strings.Contains(s, "roller"),
		strings.Contains(s, "hilltop"), strings.Contains(s, "richads"), strings.Contains(s, "evadav"),
		strings.Contains(s, "mondiad"), strings.Contains(s, "galaksion"), strings.Contains(s, "ezmob"),
		strings.Contains(s, "noviclick"), strings.Contains(s, "popcash"), strings.Contains(s, "adoperator"):
		return "push"
	default:
		return "dsp"
	}
}

func WriteJSON(path string, templates []Template) error {
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func WriteTypeScript(path string, templates []Template) error {
	return os.WriteFile(path, RenderTypeScript(templates), 0o644)
}

func RenderTypeScript(templates []Template) []byte {
	var b strings.Builder
	b.WriteString("// Code generated by cmd/codegen-traffic-templates. DO NOT EDIT.\n\n")
	b.WriteString("import type { TrafficSourceTemplate } from './traffic_source_types.js';\n\n")
	b.WriteString("export const GENERATED_TRAFFIC_SOURCE_TEMPLATES: TrafficSourceTemplate[] = [\n")
	for _, tpl := range templates {
		writeTemplateTS(&b, tpl)
	}
	b.WriteString("];\n")
	return []byte(b.String())
}

func writeTemplateTS(b *strings.Builder, tpl Template) {
	b.WriteString("  {\n")
	fmt.Fprintf(b, "    id: %q,\n", tpl.ID)
	if slug := strings.TrimSpace(tpl.BundledSlug); slug != "" {
		fmt.Fprintf(b, "    bundled_slug: %q,\n", slug)
	}
	fmt.Fprintf(b, "    name: %q,\n", tpl.Name)
	fmt.Fprintf(b, "    category: %q,\n", tpl.Category)
	if cost := strings.TrimSpace(tpl.CostSync); cost != "" {
		fmt.Fprintf(b, "    cost_sync: %q,\n", cost)
	}
	if notes := strings.TrimSpace(tpl.Notes); notes != "" {
		fmt.Fprintf(b, "    notes: %q,\n", notes)
	}
	b.WriteString("    params: [\n")
	for _, param := range tpl.Params {
		if label := strings.TrimSpace(param.Label); label != "" {
			fmt.Fprintf(b, "      { key: %q, value: %q, label: %q },\n", param.Key, param.Value, label)
		} else {
			fmt.Fprintf(b, "      { key: %q, value: %q },\n", param.Key, param.Value)
		}
	}
	b.WriteString("    ],\n")
	b.WriteString("  },\n")
}

func CountBundledTrafficSchemas() int {
	n := 0
	for _, entry := range integrationschema.BundledIntegrationTemplateCatalog {
		if entry.Category == "traffic_source" {
			n++
		}
	}
	return n
}
