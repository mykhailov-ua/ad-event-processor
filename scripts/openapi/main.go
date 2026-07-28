package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"espx/internal/openapi"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	routes, err := openapi.DocumentedRoutes(root)
	if err != nil {
		fatal(err)
	}

	outPath := filepath.Join(root, "docs", "openapi", "openapi.yaml")
	body, err := os.ReadFile(outPath)
	if err != nil {
		fatal(err)
	}
	text := string(body)
	markerStart := "# BEGIN GENERATED PATHS\n"
	markerEnd := "# END GENERATED PATHS\n"
	start := strings.Index(text, markerStart)
	end := strings.Index(text, markerEnd)
	if start < 0 || end < 0 || end <= start {
		fatal(fmt.Errorf("markers not found in %s", outPath))
	}

	var b strings.Builder
	b.WriteString(markerStart)
	byPath := make(map[string][]openapi.Route)
	for _, r := range routes {
		byPath[r.Path] = append(byPath[r.Path], r)
	}
	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		writePathOps(&b, p, byPath[p])
	}
	b.WriteString(markerEnd)

	updated := text[:start] + b.String() + text[end+len(markerEnd):]
	if err := os.WriteFile(outPath, []byte(updated), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("updated %d paths in %s\n", len(routes), outPath)
}

func writePathOps(b *strings.Builder, path string, routes []openapi.Route) {
	if !strings.HasPrefix(path, "/api/v1") {
		return
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Method < routes[j].Method
	})
	fmt.Fprintf(b, "  %s:\n", path)
	for _, r := range routes {
		writeOperation(b, r)
	}
	b.WriteString("\n")
}

func writeOperation(b *strings.Builder, r openapi.Route) {
	path := r.Path
	opID := operationID(r.Method, path)
	tag := tagFor(path)
	fmt.Fprintf(b, "    %s:\n", strings.ToLower(r.Method))
	fmt.Fprintf(b, "      tags: [%s]\n", tag)
	fmt.Fprintf(b, "      summary: %s %s\n", r.Method, path)
	fmt.Fprintf(b, "      operationId: %s\n", opID)
	switch {
	case path == "/api/v1/consent":
		fmt.Fprintf(b, "      security:\n")
		fmt.Fprintf(b, "        - ConsentHMAC: []\n")
	case path == "/api/v1/region/ingest/batch":
		fmt.Fprintf(b, "      security:\n")
		fmt.Fprintf(b, "        - AdminAPIKey: []\n")
	case path == "/api/v1/auth/login" || path == "/api/v1/auth/refresh" || path == "/api/v1/auth/logout":
		// Public cookie auth endpoints.
	default:
		fmt.Fprintf(b, "      security:\n")
		fmt.Fprintf(b, "        - AdminAPIKey: []\n")
		fmt.Fprintf(b, "        - SessionCookie: []\n")
	}
	fmt.Fprintf(b, "      responses:\n")
	switch schema := responseSchema(path, r.Method); schema {
	case "text/csv":
		fmt.Fprintf(b, "        '200':\n")
		fmt.Fprintf(b, "          description: CSV export\n")
		fmt.Fprintf(b, "          content:\n")
		fmt.Fprintf(b, "            text/csv:\n")
		fmt.Fprintf(b, "              schema:\n")
		fmt.Fprintf(b, "                type: string\n")
	case "application/pdf":
		fmt.Fprintf(b, "        '200':\n")
		fmt.Fprintf(b, "          description: PDF document\n")
		fmt.Fprintf(b, "          content:\n")
		fmt.Fprintf(b, "            application/pdf:\n")
		fmt.Fprintf(b, "              schema:\n")
		fmt.Fprintf(b, "                type: string\n")
		fmt.Fprintf(b, "                format: binary\n")
	case "204":
		fmt.Fprintf(b, "        '204':\n")
		fmt.Fprintf(b, "          description: No Content\n")
	case "":
		fmt.Fprintf(b, "        '200':\n")
		fmt.Fprintf(b, "          description: OK\n")
	default:
		fmt.Fprintf(b, "        '200':\n")
		fmt.Fprintf(b, "          description: OK\n")
		fmt.Fprintf(b, "          content:\n")
		fmt.Fprintf(b, "            application/json:\n")
		fmt.Fprintf(b, "              schema:\n")
		fmt.Fprintf(b, "                $ref: '%s'\n", schema)
	}
	if path != "/api/v1/auth/login" && path != "/api/v1/auth/refresh" {
		fmt.Fprintf(b, "        '401':\n")
		fmt.Fprintf(b, "          $ref: '#/components/responses/Unauthorized'\n")
	}
}

func operationID(method, path string) string {
	s := strings.TrimPrefix(path, "/api/v1/")
	s = strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_").Replace(s)
	return strings.ToLower(method) + "_" + s
}

func tagFor(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/")
	seg := strings.Split(rest, "/")[0]
	switch seg {
	case "auth":
		return "Auth"
	case "billing":
		return "Billing"
	case "campaigns", "forecast", "reports", "views", "dashboards":
		return "Reporting"
	case "customers", "disputes", "license", "selfserve":
		return "Customers"
	case "ops", "audit", "recon", "margin-guard":
		return "Operations"
	case "postbacks", "cost-sync":
		return "Integrations"
	case "consent":
		return "Compliance"
	case "region":
		return "MultiRegion"
	default:
		return "Management"
	}
}

func responseSchema(path, method string) string {
	if strings.HasSuffix(path, "/balance/export") {
		return "text/csv"
	}
	if strings.HasSuffix(path, "/pdf") {
		return "application/pdf"
	}
	if path == "/api/v1/consent" || path == "/api/v1/auth/logout" {
		return "204"
	}
	if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" {
		return ""
	}
	switch {
	case strings.HasSuffix(path, "/balance"):
		return "#/components/schemas/CustomerBalance"
	case strings.Contains(path, "/campaigns/{id}/stats"):
		return "#/components/schemas/CampaignStats"
	case path == "/api/v1/auth/login" || path == "/api/v1/auth/me":
		return "#/components/schemas/AuthUser"
	case path == "/api/v1/forecast/campaign":
		return "#/components/schemas/CampaignForecast"
	case strings.HasPrefix(path, "/api/v1/reports/"):
		return "#/components/schemas/ReportTable"
	case strings.HasPrefix(path, "/api/v1/dashboards/"):
		return "#/components/schemas/DashboardPayload"
	default:
		return "#/components/schemas/GenericObject"
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
