package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	bannedWord      = regexp.MustCompile(`(?i)\b(simple|elegant|clean|obviously|just|simply|nice|obvious|trivial|minimal|leverage|delve|seamless|seamlessly|moreover|furthermore|additionally|holistic|navigate|testament|harness|effortlessly|notably|essentially|basically)\b`)
	unicodeDash     = regexp.MustCompile(`[—–]`)
	gapID           = regexp.MustCompile(`(?i)\bGAP-[A-Z]+-\d+\b`)
	priorityLabel   = regexp.MustCompile(`\bP\d{2}\b`)
	milestoneTag    = regexp.MustCompile(`\bM\d+([-.][0-9A-Za-z]+)?\b`)
	milestoneWord   = regexp.MustCompile(`(?i)\bmilestone\b`)
	bannedChaosWord = regexp.MustCompile(`(?i)\bchaos\b`)
	strictNoComments = os.Getenv("STRICT_NO_COMMENTS") == "1"
	skipPrefixes    []string
)

func main() {
	skipPrefixes = loadSkipPrefixes(os.Getenv("CHECK_COMMENTS_SKIP_PREFIXES"))

	roots := []string{"internal", "pkg", "cmd", "tests", "scripts"}
	var failed bool

	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if shouldSkipDir(path) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				if skipGeneratedGo(path) {
					return nil
				}
				for _, v := range scanGoFile(path) {
					failed = true
					fmt.Fprintf(os.Stderr, "%s:%d: %s\n", path, v.line, v.msg)
				}
				return nil
			}
			if isHandwrittenQuerySQL(path) {
				for _, v := range scanSQLFile(path) {
					failed = true
					fmt.Fprintf(os.Stderr, "%s:%d: %s\n", path, v.line, v.msg)
				}
			}
			return nil
		})
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("check_comments: ok")
}

type violation struct {
	line int
	msg  string
}

func loadSkipPrefixes(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ":")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.TrimSuffix(p, "/"))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func shouldSkipDir(path string) bool {
	base := filepath.Base(path)
	if base == "db" || base == "sqlc" {
		return true
	}
	return hasSkipPrefix(path)
}

func hasSkipPrefix(path string) bool {
	clean := filepath.ToSlash(path)
	for _, prefix := range skipPrefixes {
		p := filepath.ToSlash(prefix)
		if clean == p || strings.HasPrefix(clean, p+"/") {
			return true
		}
	}
	return false
}

func isHandwrittenQuerySQL(path string) bool {
	if !strings.HasSuffix(path, ".sql") {
		return false
	}
	clean := filepath.ToSlash(path)
	if !strings.Contains(clean, "/queries/") {
		return false
	}
	return !hasSkipPrefix(path)
}

func scanGoFile(path string) []violation {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return []violation{{line: 1, msg: "parse error: " + err.Error()}}
	}

	var out []violation
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			text := commentBody(c.Text)
			if text == "" {
				if strictNoComments {
					pos := fset.Position(c.Pos())
					out = append(out, violation{line: pos.Line, msg: "empty comment forbidden (R9.3)"})
				}
				continue
			}
			pos := fset.Position(c.Pos())
			if v := checkCommentText(text); v != "" {
				out = append(out, violation{line: pos.Line, msg: v})
			}
		}
	}
	return out
}

func scanSQLFile(path string) []violation {
	f, err := os.Open(path)
	if err != nil {
		return []violation{{line: 1, msg: "read error: " + err.Error()}}
	}
	defer f.Close()

	var out []violation
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		text := sqlCommentBody(sc.Text())
		if text == "" {
			continue
		}
		if v := checkCommentText(text); v != "" {
			out = append(out, violation{line: lineNo, msg: v})
		}
	}
	if err := sc.Err(); err != nil {
		return []violation{{line: 1, msg: "read error: " + err.Error()}}
	}
	return out
}

func sqlCommentBody(line string) string {
	idx := strings.Index(line, "--")
	if idx < 0 {
		return ""
	}
	text := strings.TrimSpace(line[idx+2:])
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "name:") {
		return ""
	}
	return text
}

func commentBody(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	text = strings.TrimSpace(strings.TrimPrefix(text, "*"))
	return text
}

func checkCommentText(text string) string {
	if strictNoComments && !isAllowedDirective(text) {
		return "comment forbidden: only //go: and //nolint: allowed (R9.3)"
	}
	if gapID.MatchString(text) {
		return "GAP-* ID in comment (forbidden; use GAP_SPECS.md)"
	}
	if priorityLabel.MatchString(text) {
		return "P## priority in comment (forbidden)"
	}
	if milestoneTag.MatchString(text) {
		return "milestone tag (M*) in comment (forbidden)"
	}
	if milestoneWord.MatchString(text) {
		return "word 'milestone' in comment (forbidden)"
	}
	if bannedChaosWord.MatchString(text) {
		return "word 'chaos' in comment (use fault/resilience naming)"
	}
	if bannedWord.MatchString(text) {
		return "banned word in comment"
	}
	if unicodeDash.MatchString(text) {
		return "unicode dash in comment (use ASCII -)"
	}
	for _, r := range text {
		if r < 0x20 || r > 0x7e {
			if r != '\t' && !unicode.IsSpace(r) {
				return "non-ASCII in comment"
			}
		}
	}
	return ""
}

func isAllowedDirective(text string) bool {
	t := strings.TrimSpace(text)
	return strings.HasPrefix(t, "go:") || strings.HasPrefix(t, "nolint:")
}

func skipGeneratedGo(path string) bool {
	base := filepath.Base(path)
	switch {
	case strings.HasSuffix(base, ".pb.go"),
		strings.HasSuffix(base, "_grpc.pb.go"),
		strings.HasSuffix(base, "_vtproto.pb.go"),
		strings.HasSuffix(base, "_bpfel.go"),
		strings.HasSuffix(base, "_bpfeb.go"):
		return true
	}
	return false
}
