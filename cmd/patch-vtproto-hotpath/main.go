// patch-vtproto-hotpath entrypoint. Package documentation: doc.go.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const defaultPath = "internal/ingest/pb/events_vtproto.pb.go"

var patches = []struct {
	from string
	to   string
}{
	{
		from: "m.ExtraKeys = append(m.ExtraKeys, make([]byte, postIndex-iNdEx))\n\t\t\tcopy(m.ExtraKeys[len(m.ExtraKeys)-1], dAtA[iNdEx:postIndex])",
		to:   "m.ExtraKeys = appendReuseBytes(m.ExtraKeys, dAtA[iNdEx:postIndex])",
	},
	{
		from: "m.ExtraValues = append(m.ExtraValues, make([]byte, postIndex-iNdEx))\n\t\t\tcopy(m.ExtraValues[len(m.ExtraValues)-1], dAtA[iNdEx:postIndex])",
		to:   "m.ExtraValues = appendReuseBytes(m.ExtraValues, dAtA[iNdEx:postIndex])",
	},
}

var nilSliceGuardRE = regexp.MustCompile(`\n\t\t\tif m\.[A-Za-z0-9]+ == nil \{\n\t\t\t\tm\.[A-Za-z0-9]+ = \[\]byte\{\}\n\t\t\t\}`)

func main() {
	path := defaultPath
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "patch_vtproto_hotpath: read %s: %v\n", path, err)
		os.Exit(1)
	}
	text := string(data)
	changed := false

	if !strings.Contains(text, "appendReuseBytes(m.ExtraKeys") {
		for _, p := range patches {
			if !strings.Contains(text, p.from) {
				fmt.Fprintf(os.Stderr, "patch_vtproto_hotpath: pattern missing in %s (buf plugin output changed?)\n", path)
				os.Exit(1)
			}
			text = strings.Replace(text, p.from, p.to, 1)
		}
		changed = true
	}

	stripped := nilSliceGuardRE.ReplaceAllString(text, "")
	if stripped != text {
		text = stripped
		changed = true
	}

	if !changed {
		return
	}

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "patch_vtproto_hotpath: write %s: %v\n", path, err)
		os.Exit(1)
	}
}
