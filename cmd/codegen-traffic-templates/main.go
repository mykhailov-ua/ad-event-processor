// codegen-traffic-templates entrypoint. Package documentation: doc.go.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/traffictemplates"
)

func main() {
	check := flag.Bool("check", false, "exit 1 when generated output is stale")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	schemasDir := integrationschema.SchemaRootDir()
	sidecarPath := filepath.Join(*root, "deploy", "vendor", "traffic_source_ui.yaml")
	outPath := filepath.Join(*root, "internal", "traffictemplates", "generated_templates.json")

	templates, err := traffictemplates.Generate(schemasDir, sidecarPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "codegen-traffic-templates: %v\n", err)
		os.Exit(1)
	}

	catalogCount := traffictemplates.CountBundledTrafficSchemas()
	slugSeen := map[string]struct{}{}
	for _, tpl := range templates {
		if slug := tpl.BundledSlug; slug != "" {
			slugSeen[slug] = struct{}{}
		}
	}
	if len(slugSeen) < catalogCount {
		fmt.Fprintf(os.Stderr, "codegen-traffic-templates: bundled slug coverage %d < catalog %d\n", len(slugSeen), catalogCount)
		os.Exit(1)
	}

	rendered, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "codegen-traffic-templates: %v\n", err)
		os.Exit(1)
	}
	rendered = append(rendered, '\n')

	if *check {
		existing, err := os.ReadFile(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "codegen-traffic-templates: read %s: %v\n", outPath, err)
			os.Exit(1)
		}
		if string(existing) != string(rendered) {
			fmt.Fprintf(os.Stderr, "codegen-traffic-templates: stale %s (run go run ./cmd/codegen-traffic-templates)\n", outPath)
			os.Exit(1)
		}
		_, _ = fmt.Fprintf(os.Stdout, "codegen-traffic-templates: OK templates=%d catalog=%d\n", len(templates), catalogCount)
		return
	}

	if err := os.WriteFile(outPath, rendered, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "codegen-traffic-templates: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "codegen-traffic-templates: wrote %s (%d templates, %d catalog schemas)\n", outPath, len(templates), catalogCount)
}
