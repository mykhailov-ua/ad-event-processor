package main

import (
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"espx/pkg/commentkeep"
)

var (
	write   = flag.Bool("write", false, "rewrite files in place (default: dry-run)")
	verbose = flag.Bool("v", false, "print each changed file")
)

var (
	roots = []string{"internal", "pkg", "cmd", "tests", "scripts", "deploy", "api", ".github"}
	extra = []string{
		"Makefile",
		"docker-compose.yaml",
		"docker-compose.load-test.yaml",
		"deploy/compose/docker-compose.yaml",
		"deploy/compose/docker-compose.load-test.yaml",
		"sqlc.yaml",
		".golangci.yaml",
		"lefthook.yaml",
		"buf.yaml",
		"Taskfile.yaml",
		".env.example",
	}
)

func main() {
	flag.Parse()

	allExtra := append([]string{}, extra...)
	allExtra = append(allExtra, discoverRootDockerfiles()...)

	var changed, skipped, errors int
	walkRoots := append([]string{}, roots...)
	for _, root := range walkRoots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipDir(path, d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			n, stripErr, ok := processPath(path)
			if !ok {
				skipped++
				return nil
			}
			if stripErr != nil {
				errors++
				fmt.Fprintf(os.Stderr, "%s: %v\n", path, stripErr)
				return nil
			}
			if n > 0 {
				changed++
				if *verbose {
					fmt.Printf("%s: stripped %d comment(s)\n", path, n)
				}
			}
			return nil
		})
	}
	for _, path := range allExtra {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		n, stripErr, ok := processPath(path)
		if !ok {
			skipped++
			continue
		}
		if stripErr != nil {
			errors++
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, stripErr)
			continue
		}
		if n > 0 {
			changed++
			if *verbose {
				fmt.Printf("%s: stripped %d comment(s)\n", path, n)
			}
		}
	}

	mode := "dry-run"
	if *write {
		mode = "write"
	}
	fmt.Printf("strip_comments: %s changed=%d skipped=%d errors=%d\n", mode, changed, skipped, errors)
	if errors > 0 {
		os.Exit(1)
	}
}

func processPath(path string) (int, error, bool) {
	if skipFile(path) {
		return 0, nil, false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		n, err := processGoFile(path)
		return n, err, true
	default:
		if lang, ok := langForPath(path); ok {
			n, err := processTextFile(path, lang)
			return n, err, true
		}
		return 0, nil, false
	}
}

func skipDir(path, base string) bool {
	switch base {
	case "db", "sqlc", "gen", "node_modules", "vendor", ".git":
		return true
	}
	return strings.Contains(path, string(filepath.Separator)+"api"+string(filepath.Separator)+"gen"+string(filepath.Separator))
}

func skipFile(path string) bool {
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

func processGoFile(path string) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return 0, err
	}

	file := fset.File(f.Pos())
	var removed int
	ranges := make([][2]int, 0, len(f.Comments))
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			body := commentBody(c.Text)
			if isAllowedDirective(body) {
				continue
			}
			start := file.Offset(c.Pos())
			end := file.Offset(c.End())
			pos := file.Position(c.Pos())
			lineStart := int(file.LineStart(pos.Line))
			if lineStart <= start {
				prefix := strings.TrimSpace(string(src[lineStart:start]))
				if prefix == "" && end < len(src) && src[end] == '\n' {
					end++
				}
			}
			ranges = append(ranges, [2]int{start, end})
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}

	ranges = mergeRanges(ranges)
	out := deleteRanges(src, ranges)
	out, err = format.Source(out)
	if err != nil {
		return 0, fmt.Errorf("format: %w", err)
	}

	if !*write {
		return removed, nil
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return 0, err
	}
	return removed, nil
}

func deleteRanges(src []byte, ranges [][2]int) []byte {
	for i := len(ranges) - 1; i >= 0; i-- {
		start, end := ranges[i][0], ranges[i][1]
		if start < 0 || end > len(src) || start >= end {
			continue
		}
		src = append(src[:start], src[end:]...)
	}
	return src
}

func mergeRanges(ranges [][2]int) [][2]int {
	if len(ranges) == 0 {
		return nil
	}
	out := append([][2]int(nil), ranges...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j][0] < out[i][0] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	merged := make([][2]int, 0, len(out))
	cur := out[0]
	for i := 1; i < len(out); i++ {
		if out[i][0] <= cur[1] {
			if out[i][1] > cur[1] {
				cur[1] = out[i][1]
			}
			continue
		}
		if cur[0] < cur[1] {
			merged = append(merged, cur)
		}
		cur = out[i]
	}
	if cur[0] < cur[1] {
		merged = append(merged, cur)
	}
	return merged
}

func commentBody(raw string) string {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "//")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	text = strings.TrimSpace(strings.TrimPrefix(text, "*"))
	return text
}

func isAllowedDirective(text string) bool {
	return commentkeep.Keep(text)
}

func discoverRootDockerfiles() []string {
	matches, err := filepath.Glob("deploy/docker/Dockerfile*")
	if err != nil {
		return nil
	}
	return matches
}
