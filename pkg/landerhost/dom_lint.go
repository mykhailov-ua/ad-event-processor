package landerhost

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const DefaultCSPPolicy = "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'"

const (
	DomLintRuleMetaRefresh      = "meta_refresh"
	DomLintRuleHiddenOverlay    = "hidden_overlay"
	DomLintRuleOpacityClickTrap = "opacity_click_trap"
	DomLintRuleLocationChain    = "location_chain"
)

type DomLintViolation struct {
	Path   string
	Rule   string
	Detail string
}

type DomLintResult struct {
	Violations []DomLintViolation
}

func (r DomLintResult) OK() bool {
	return len(r.Violations) == 0
}

type DomLintError struct {
	Result DomLintResult
}

func (e *DomLintError) Error() string {
	if e == nil || len(e.Result.Violations) == 0 {
		return "lander dom lint failed"
	}
	v := e.Result.Violations[0]
	return fmt.Sprintf("lander dom lint: %s in %s (%s)", v.Rule, v.Path, v.Detail)
}

var (
	metaRefreshPattern      = regexp.MustCompile(`(?is)<meta[^>]+http-equiv\s*=\s*['"]?refresh`)
	hiddenOverlayPattern    = regexp.MustCompile(`(?is)display\s*:\s*none[^}]{0,400}(position\s*:\s*(?:fixed|absolute))[^}]{0,400}((?:100vh|100vw|100%))`)
	opacityClickTrapPattern = regexp.MustCompile(`(?is)opacity\s*:\s*0[^}]{0,300}position\s*:\s*(?:fixed|absolute)`)
	locationAssignPattern   = regexp.MustCompile(`(?i)(?:window\.)?location(?:\.href)?\s*=`)
	locationReplacePattern  = regexp.MustCompile(`(?i)(?:window\.)?location\.replace\s*\(`)
)

func LintAsset(path string, content []byte) []DomLintViolation {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".html"), strings.HasSuffix(name, ".htm"):
		return lintHTML(path, content)
	case strings.HasSuffix(name, ".css"):
		return lintCSS(path, content)
	case strings.HasSuffix(name, ".js"):
		return lintJS(path, content)
	default:
		return nil
	}
}

func lintHTML(path string, content []byte) []DomLintViolation {
	body := string(content)
	var out []DomLintViolation
	if metaRefreshPattern.MatchString(body) {
		out = append(out, DomLintViolation{
			Path:   path,
			Rule:   DomLintRuleMetaRefresh,
			Detail: "meta http-equiv refresh is banned on production landers",
		})
	}
	if hiddenOverlayPattern.MatchString(body) {
		out = append(out, DomLintViolation{
			Path:   path,
			Rule:   DomLintRuleHiddenOverlay,
			Detail: "full-viewport display:none overlay pattern",
		})
	}
	if opacityClickTrapPattern.MatchString(body) {
		out = append(out, DomLintViolation{
			Path:   path,
			Rule:   DomLintRuleOpacityClickTrap,
			Detail: "opacity:0 positioned click target",
		})
	}
	out = append(out, lintJS(path, content)...)
	return out
}

func lintCSS(path string, content []byte) []DomLintViolation {
	body := string(content)
	var out []DomLintViolation
	if hiddenOverlayPattern.MatchString(body) {
		out = append(out, DomLintViolation{
			Path:   path,
			Rule:   DomLintRuleHiddenOverlay,
			Detail: "full-viewport display:none overlay pattern",
		})
	}
	if opacityClickTrapPattern.MatchString(body) {
		out = append(out, DomLintViolation{
			Path:   path,
			Rule:   DomLintRuleOpacityClickTrap,
			Detail: "opacity:0 positioned click target",
		})
	}
	return out
}

func lintJS(path string, content []byte) []DomLintViolation {
	body := string(content)
	assigns := len(locationAssignPattern.FindAllString(body, -1))
	replaces := len(locationReplacePattern.FindAllString(body, -1))
	if assigns+replaces >= 2 {
		return []DomLintViolation{{
			Path:   path,
			Rule:   DomLintRuleLocationChain,
			Detail: "nested window.location redirect chain",
		}}
	}
	return nil
}

func (st *Store) LintVersion(landerID uuid.UUID, version int) (DomLintResult, error) {
	if st == nil {
		return DomLintResult{}, fmt.Errorf("lander store unavailable")
	}
	if landerID == uuid.Nil || version <= 0 {
		return DomLintResult{}, fmt.Errorf("invalid lander version")
	}
	files, err := st.ListVersionFiles(landerID, version)
	if err != nil {
		return DomLintResult{}, err
	}
	result := DomLintResult{}
	for _, ent := range files {
		if !IsEditableTextPath(ent.Path) {
			continue
		}
		raw, err := st.ReadVersionFile(landerID, version, ent.Path)
		if err != nil {
			return DomLintResult{}, err
		}
		violations := LintAsset(ent.Path, raw)
		result.Violations = append(result.Violations, violations...)
	}
	return result, nil
}
