package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Route is one HTTP route on the management JSON API.
type Route struct {
	Method string
	Path   string
}

// Key returns method and path joined for set lookups.
func (r Route) Key() string {
	return r.Method + " " + r.Path
}

var (
	handleFuncRE = regexp.MustCompile(`HandleFunc\("(GET|POST|PUT|DELETE|PATCH) (/api/v1[^"]+)"`)
	literalRE    = regexp.MustCompile(`\{"(GET|POST|PUT|DELETE|PATCH) (/api/v1[^"]+)"`)
)

// DiscoverAPIV1Routes scans management and adminapi handler registrations.
func DiscoverAPIV1Routes(repoRoot string) ([]Route, error) {
	dirs := []string{
		filepath.Join(repoRoot, "internal", "management"),
		filepath.Join(repoRoot, "internal", "adminapi"),
	}
	seen := make(map[string]Route)
	for _, dir := range dirs {
		if err := discoverDir(dir, seen); err != nil {
			return nil, err
		}
	}
	out := make([]Route, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func discoverDir(dir string, seen map[string]Route) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		text := string(body)
		for _, m := range handleFuncRE.FindAllStringSubmatch(text, -1) {
			addRoute(seen, m[1], m[2])
		}
		for _, m := range literalRE.FindAllStringSubmatch(text, -1) {
			addRoute(seen, m[1], m[2])
		}
		return nil
	})
}

func addRoute(seen map[string]Route, method, path string) {
	r := Route{Method: method, Path: path}
	seen[r.Key()] = r
}

// DocumentedRoutes returns discovered routes minus 501 stubs.
func DocumentedRoutes(repoRoot string) ([]Route, error) {
	all, err := DiscoverAPIV1Routes(repoRoot)
	if err != nil {
		return nil, err
	}
	out := make([]Route, 0, len(all))
	for _, r := range all {
		if IsStub(r.Method, r.Path) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
