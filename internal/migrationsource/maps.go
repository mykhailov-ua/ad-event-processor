package migrationsource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ad-event-processor/internal/config"

	"gopkg.in/yaml.v3"
)

// MacroEntry maps a foreign tracker token to an internal click query key.
type MacroEntry struct {
	Source      string `yaml:"source"`
	TargetKey   string `yaml:"target_key"`
	Passthrough bool   `yaml:"passthrough,omitempty"`
	IngressCost bool   `yaml:"ingress_cost,omitempty"`
}

// SourceEntry maps a foreign traffic source label to bundled integration slug.
type SourceEntry struct {
	KeitaroName  string `yaml:"keitaro_name"`
	BinomName    string `yaml:"binom_name"`
	BundledSlug  string `yaml:"bundled_slug"`
	UITemplateID string `yaml:"ui_template_id"`
}

type macrosFile struct {
	Version int          `yaml:"version"`
	Macros  []MacroEntry `yaml:"macros"`
}

type sourcesFile struct {
	Version int           `yaml:"version"`
	Sources []SourceEntry `yaml:"sources"`
}

// Maps holds loaded macro and traffic-source mapping tables.
type Maps struct {
	KeitaroMacros  []MacroEntry
	KeitaroSources []SourceEntry
	BinomMacros    []MacroEntry
	BinomSources   []SourceEntry
}

// MapsRootDir resolves deploy/vendor/migration for dev and install roots.
func MapsRootDir() string {
	if root := config.InstallRootFromEnv(); root != "" {
		p := filepath.Join(root, "deploy", "vendor", "migration")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	candidates := []string{
		filepath.Join("deploy", "vendor", "migration"),
		filepath.Join("..", "..", "deploy", "vendor", "migration"),
		filepath.Join("..", "..", "..", "deploy", "vendor", "migration"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return candidates[0]
}

// LoadMaps reads YAML tables from dir.
func LoadMaps(dir string) (*Maps, error) {
	if strings.TrimSpace(dir) == "" {
		dir = MapsRootDir()
	}
	out := &Maps{}
	if err := loadMacrosFile(filepath.Join(dir, "keitaro_macros.yaml"), &out.KeitaroMacros); err != nil {
		return nil, fmt.Errorf("keitaro_macros: %w", err)
	}
	if err := loadMacrosFile(filepath.Join(dir, "binom_macros.yaml"), &out.BinomMacros); err != nil {
		return nil, fmt.Errorf("binom_macros: %w", err)
	}
	keitaroSrc, err := loadSourcesFile(filepath.Join(dir, "keitaro_sources.yaml"))
	if err != nil {
		return nil, fmt.Errorf("keitaro_sources: %w", err)
	}
	out.KeitaroSources = keitaroSrc
	binomSrc, err := loadSourcesFile(filepath.Join(dir, "binom_sources.yaml"))
	if err != nil {
		return nil, fmt.Errorf("binom_sources: %w", err)
	}
	out.BinomSources = binomSrc
	if err := validateMacroEntries(out.KeitaroMacros); err != nil {
		return nil, fmt.Errorf("keitaro_macros validate: %w", err)
	}
	if err := validateMacroEntries(out.BinomMacros); err != nil {
		return nil, fmt.Errorf("binom_macros validate: %w", err)
	}
	return out, nil
}

func loadMacrosFile(path string, dest *[]MacroEntry) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc macrosFile
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if doc.Version <= 0 {
		return fmt.Errorf("version must be positive")
	}
	*dest = doc.Macros
	return nil
}

func loadSourcesFile(path string) ([]SourceEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc sourcesFile
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Version <= 0 {
		return nil, fmt.Errorf("version must be positive")
	}
	return doc.Sources, nil
}

func validateMacroEntries(entries []MacroEntry) error {
	seen := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		src := strings.TrimSpace(e.Source)
		if src == "" {
			return fmt.Errorf("macro source is required")
		}
		if _, ok := seen[src]; ok {
			return fmt.Errorf("duplicate macro source %q", src)
		}
		seen[src] = struct{}{}
		key := strings.TrimSpace(e.TargetKey)
		if key == "" && !e.Passthrough {
			return fmt.Errorf("macro %q missing target_key", src)
		}
	}
	return nil
}
