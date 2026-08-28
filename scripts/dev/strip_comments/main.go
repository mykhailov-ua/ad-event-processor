package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	var goOK, goFail, shOK, luaOK int
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", "var", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(path) {
			return nil
		}

		switch {
		case strings.HasSuffix(path, ".go"):
			if err := stripGoFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "go %s: %v\n", path, err)
				goFail++
				return nil
			}
			goOK++
		case strings.HasSuffix(path, ".sh"):
			if err := stripShellFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "sh %s: %v\n", path, err)
				return nil
			}
			shOK++
		case strings.HasSuffix(path, ".lua"):
			if err := stripLuaFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "lua %s: %v\n", path, err)
				return nil
			}
			luaOK++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "stripped go=%d fail=%d sh=%d lua=%d\n", goOK, goFail, shOK, luaOK)
	if goFail > 0 {
		os.Exit(1)
	}
}

func shouldSkipFile(path string) bool {
	base := filepath.Base(path)
	if base == ".env.example" || base == ".gitignore" || base == ".gitkeep" {
		return true
	}
	if strings.HasSuffix(path, "/.env.example") || strings.HasSuffix(path, "/.gitignore") {
		return true
	}
	if strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
		return true
	}
	return false
}

func keepGoComment(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "//") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "//"))
	} else if strings.HasPrefix(t, "/*") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "/*"))
		t = strings.TrimSuffix(t, "*/")
		t = strings.TrimSpace(t)
	}
	prefixes := []string{
		"go:",
		"+build",
		"line ",
		"export ",
		"garble:",
		"nolint",
		"lint:",
		"staticcheck:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

func stripGoFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return err
	}
	stripGoAST(f)
	var out bytes.Buffer
	if err := format.Node(&out, fset, f); err != nil {
		return err
	}
	outBytes := out.Bytes()
	if !bytes.HasSuffix(outBytes, []byte("\n")) {
		outBytes = append(outBytes, '\n')
	}
	return os.WriteFile(path, outBytes, 0o644)
}

func stripGoAST(f *ast.File) {
	f.Doc = nil
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			x.Doc = nil
		case *ast.FuncDecl:
			x.Doc = nil
		case *ast.Field:
			x.Doc = nil
		case *ast.TypeSpec:
			x.Doc = nil
		case *ast.ValueSpec:
			x.Doc = nil
		case *ast.File:
			return true
		}
		return true
	})

	var kept []*ast.CommentGroup
	for _, cg := range f.Comments {
		var list []*ast.Comment
		for _, c := range cg.List {
			if keepGoComment(c.Text) {
				list = append(list, c)
			}
		}
		if len(list) > 0 {
			kept = append(kept, &ast.CommentGroup{List: list})
		}
	}
	f.Comments = kept
}

func stripShellFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if i == 0 && strings.HasPrefix(trim, "#!") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trim, "# shellcheck") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		out = append(out, line)
	}
	body := strings.Join(out, "\n")
	if len(src) > 0 && src[len(src)-1] == '\n' && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func stripLuaFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(src), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if stripped := stripLuaLineComment(line); stripped != "" || strings.TrimSpace(line) == "" {
			out = append(out, stripped)
		}
	}
	body := strings.Join(out, "\n")
	if len(src) > 0 && src[len(src)-1] == '\n' && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func stripLuaLineComment(line string) string {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line)-1; i++ {
		if line[i] == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if line[i] == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && line[i] == '-' && line[i+1] == '-' {
			return strings.TrimRight(line[:i], " \t")
		}
	}
	return line
}
