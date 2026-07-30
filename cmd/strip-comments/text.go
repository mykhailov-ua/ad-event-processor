package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

type textLang int

const (
	langLua textLang = iota
	langSQL
	langShell
	langYAML
	langMake
	langC
	langProto
)

func processTextFile(path string, lang textLang) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var out []byte
	var removed int
	switch lang {
	case langC:
		out, removed = stripC(src)
	default:
		out, removed = stripLineOriented(src, lang)
	}
	if removed == 0 {
		return 0, nil
	}
	if !*write {
		return removed, nil
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return 0, err
	}
	return removed, nil
}

func stripLineOriented(src []byte, lang textLang) ([]byte, int) {
	lines := bytes.SplitAfter(src, []byte("\n"))
	var removed int
	out := make([]byte, 0, len(src))
	for _, line := range lines {
		stripped, n, drop := stripOneLine(line, lang)
		removed += n
		if !drop {
			out = append(out, stripped...)
		}
	}
	return out, removed
}

func stripOneLine(line []byte, lang textLang) ([]byte, int, bool) {
	if len(line) == 0 {
		return line, 0, false
	}
	if lang == langShell && bytes.HasPrefix(line, []byte("#!")) {
		return line, 0, false
	}

	marker, width := lineCommentMarker(lang)
	if marker == 0 {
		return line, 0, false
	}

	idx := findCommentIndex(line, marker, width, lang)
	if idx < 0 {
		return line, 0, false
	}

	body := commentLineBody(line[idx:], lang)
	if keepTextComment(body, lang) {
		return line, 0, false
	}

	prefix := bytes.TrimRight(line[:idx], " \t")
	if len(prefix) == 0 {
		return nil, 1, true
	}
	out := append(append([]byte{}, prefix...), '\n')
	return out, 1, false
}

func lineCommentMarker(lang textLang) (byte, int) {
	switch lang {
	case langLua, langSQL:
		return '-', 2
	case langShell, langYAML, langMake:
		return '#', 1
	case langProto:
		return '/', 2
	default:
		return 0, 0
	}
}

func commentLineBody(line []byte, lang textLang) string {
	s := strings.TrimSpace(string(line))
	switch lang {
	case langLua, langSQL:
		s = strings.TrimPrefix(s, "--")
	case langShell, langYAML, langMake:
		s = strings.TrimPrefix(s, "#")
	case langProto:
		s = strings.TrimPrefix(s, "//")
	}
	return strings.TrimSpace(s)
}

func keepTextComment(body string, lang textLang) bool {
	switch lang {
	case langSQL:
		if strings.HasPrefix(body, "+goose") {
			return true
		}
		if strings.HasPrefix(body, "name:") {
			return true
		}
	case langYAML:
		if strings.HasPrefix(body, "BEGIN GENERATED") || strings.HasPrefix(body, "END GENERATED") {
			return true
		}
	}
	return false
}

func findCommentIndex(line []byte, marker byte, width int, lang textLang) int {
	inSingle := false
	inDouble := false
	for i := 0; i < len(line); i++ {
		if inDouble {
			if line[i] == '\\' && i+1 < len(line) {
				i++
				continue
			}
			if line[i] == '"' {
				inDouble = false
			}
			continue
		}
		if inSingle {
			if line[i] == '\\' && i+1 < len(line) {
				i++
				continue
			}
			if line[i] == '\'' {
				inSingle = false
			}
			continue
		}
		if lang == langLua && i+1 < len(line) && line[i] == '[' && line[i+1] == '[' {
			end := bytes.Index(line[i+2:], []byte("]]"))
			if end >= 0 {
				i += 2 + end + 1
				continue
			}
		}
		if line[i] == '"' {
			inDouble = true
			continue
		}
		if line[i] == '\'' {
			inSingle = true
			continue
		}
		if line[i] == marker && i+width <= len(line) {
			ok := true
			for j := 1; j < width; j++ {
				if line[i+j] != marker {
					ok = false
					break
				}
			}
			if ok {
				return i
			}
		}
	}
	return -1
}

func stripC(src []byte) ([]byte, int) {
	var out bytes.Buffer
	removed := 0
	inString := byte(0)
	i := 0
	for i < len(src) {
		if inString != 0 {
			out.WriteByte(src[i])
			if src[i] == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
			} else if src[i] == inString {
				inString = 0
			}
			i++
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			removed++
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			removed++
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if src[i] == '"' || src[i] == '\'' {
			inString = src[i]
		}
		out.WriteByte(src[i])
		i++
	}
	return out.Bytes(), removed
}

func langForPath(path string) (textLang, bool) {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case strings.HasPrefix(base, ".env") || strings.HasSuffix(base, ".env"):
		return langYAML, true
	case strings.HasPrefix(base, "dockerfile"), base == "containerfile":
		return langYAML, true
	case ext == ".conf", ext == ".ini", ext == ".toml":
		return langYAML, true
	case ext == ".lua":
		return langLua, true
	case ext == ".sql":
		return langSQL, true
	case ext == ".sh", ext == ".bash":
		return langShell, true
	case ext == ".c", ext == ".h":
		return langC, true
	case ext == ".yaml", ext == ".yml":
		return langYAML, true
	case ext == ".proto":
		return langProto, true
	case base == "makefile" || ext == ".mk":
		return langMake, true
	default:
		return 0, false
	}
}
