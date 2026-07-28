package openapi

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is a parsed OpenAPI 3 document (paths + security schemes).
type Document struct {
	OpenAPI    string         `yaml:"openapi"`
	Paths      map[string]any `yaml:"paths"`
	Components struct {
		SecuritySchemes map[string]any `yaml:"securitySchemes"`
		Schemas         map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

// LoadSpec parses docs/openapi/openapi.yaml from repoRoot.
func LoadSpec(repoRoot string) (*Document, error) {
	path := strings.Join([]string{repoRoot, "docs", "openapi", "openapi.yaml"}, string(os.PathSeparator))
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read openapi spec: %w", err)
	}
	var doc Document
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi spec: %w", err)
	}
	if doc.OpenAPI == "" {
		return nil, fmt.Errorf("openapi version missing")
	}
	if len(doc.Paths) == 0 {
		return nil, fmt.Errorf("openapi paths missing")
	}
	return &doc, nil
}

// SpecRoutes lists method+path pairs declared in the OpenAPI document.
func SpecRoutes(doc *Document) []Route {
	if doc == nil {
		return nil
	}
	var out []Route
	for path, raw := range doc.Paths {
		methods, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for method := range methods {
			switch strings.ToLower(method) {
			case "get", "post", "put", "delete", "patch":
				out = append(out, Route{Method: strings.ToUpper(method), Path: path})
			}
		}
	}
	sortRoutes(out)
	return out
}

func sortRoutes(routes []Route) {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
}

// OpenAPIPath converts Go 1.22 path patterns to OpenAPI path templates.
func OpenAPIPath(goPath string) string {
	return goPath
}

// HasAdminHTMLPaths reports forbidden /admin/* entries in the spec.
func HasAdminHTMLPaths(doc *Document) []string {
	if doc == nil {
		return nil
	}
	var bad []string
	for path := range doc.Paths {
		if strings.HasPrefix(path, "/admin/") {
			bad = append(bad, path)
		}
	}
	return bad
}

// SecuritySchemeNames returns declared security scheme keys.
func SecuritySchemeNames(doc *Document) []string {
	if doc == nil || doc.Components.SecuritySchemes == nil {
		return nil
	}
	names := make([]string, 0, len(doc.Components.SecuritySchemes))
	for name := range doc.Components.SecuritySchemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
