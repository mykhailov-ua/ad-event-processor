package management

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainRegistry_AllProductionFilesMapped(t *testing.T) {
	dir := domainSourceDir(t)
	var unmapped []string
	for _, name := range listProductionGoFiles(dir) {
		if FileDomain(name) == "" {
			unmapped = append(unmapped, name)
		}
	}
	require.Empty(t, unmapped, "unmapped production files: %v", unmapped)
}

func TestDomainRegistry_EachDomainHasTestFile(t *testing.T) {
	dir := domainSourceDir(t)
	testFiles := listTestGoFiles(dir)
	for _, d := range ManagementDomains {
		found := false
		for _, tf := range testFiles {
			if d.hasTestFile(tf) {
				found = true
				break
			}
		}
		require.True(t, found, "domain %q has no matching _test.go", d.ID)
	}
}

func TestDomainRegistry_NoImportCycles(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "list", "-e", "-json", "./internal/management/...")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.NotContains(t, string(out), "import cycle")
}

func TestBoundaryDTO_HandlersSQLCAllowlist(t *testing.T) {
	dir := domainSourceDir(t)
	fset := token.NewFileSet()
	for _, name := range listProductionGoFiles(dir) {
		if !strings.HasPrefix(name, "handler") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		importsSQLC := false
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if strings.HasSuffix(p, "/ingestion/sqlc") {
				importsSQLC = true
				break
			}
		}
		if !importsSQLC {
			continue
		}
		assert.True(t, handlerSQLCAllowlist[name],
			"%s imports sqlc directly; move DB access to service_* or add to allowlist during migration", name)
	}
}

func TestBoundaryDTO_ServiceFilesNoHTTPResponse(t *testing.T) {
	dir := domainSourceDir(t)
	fset := token.NewFileSet()
	for _, name := range listProductionGoFiles(dir) {
		if !strings.HasPrefix(name, "service_") {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || ident.Name != "http" {
				return true
			}
			if sel.Sel.Name == "ResponseWriter" || sel.Sel.Name == "HandlerFunc" {
				assert.Fail(t, "service file must not reference http transport types", "file=%s", name)
			}
			return true
		})
	}
}

func TestDomainBusinessLogicCoverage(t *testing.T) {
	profile := os.Getenv("ESPX_MGMT_COVER_PROFILE")
	if profile == "" {
		t.Skip("set ESPX_MGMT_COVER_PROFILE (see scripts/ci/management_domain_coverage.sh)")
	}
	covered, total := parseCoverProfileByDomain(t, profile)
	for _, d := range ManagementDomains {
		if len(d.LogicFiles) == 0 {
			continue
		}
		c, n := covered[d.ID], total[d.ID]
		require.Greater(t, n, 0, "domain %q has LogicFiles but no coverable statements in profile", d.ID)
		ratio := float64(c) / float64(n)
		assert.GreaterOrEqual(t, ratio, domainBusinessLogicCoverageMin,
			"domain %q logic-file coverage %.1f%% < %.0f%% (%d/%d stmts)",
			d.ID, ratio*100, domainBusinessLogicCoverageMin*100, c, n)
	}
}

func domainSourceDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	if filepath.Base(wd) == "management" {
		return wd
	}
	return filepath.Join(repoRoot(t), "internal", "management")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd
	}
	return filepath.Join(wd, "..", "..")
}

func listProductionGoFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

func listTestGoFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

func parseCoverProfileByDomain(t *testing.T, path string) (covered, total map[string]int) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	covered = make(map[string]int)
	total = make(map[string]int)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		filePart := parts[0]
		if !strings.Contains(filePart, "/internal/management/") {
			continue
		}
		name := filepath.Base(strings.Split(filePart, ":")[0])
		domain := FileDomain(name)
		if domain == "" {
			continue
		}
		logic := domainLogicFiles(domain)
		if logic == nil || !logic[name] {
			continue
		}
		stmts, err := strconv.Atoi(parts[1])
		require.NoError(t, err)
		hits, err := strconv.Atoi(parts[2])
		require.NoError(t, err)
		total[domain] += stmts
		covered[domain] += hits
	}
	require.NoError(t, sc.Err())
	return covered, total
}
