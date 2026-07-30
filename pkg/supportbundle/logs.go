package supportbundle

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func tailLogLines(dir string, maxLines int) ([]string, error) {
	if maxLines <= 0 {
		maxLines = 10_000
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".json") {
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)

	var lines []string
	for _, path := range files {
		more, err := readFileLines(path)
		if err != nil {
			continue
		}
		lines = append(lines, more...)
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}

func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil && err != io.EOF {
		return lines, err
	}
	return lines, nil
}
