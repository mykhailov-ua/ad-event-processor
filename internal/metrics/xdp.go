package metrics

import "os"

func RecordXDPPinnedMapCount(pinDir string) {
	entries, err := os.ReadDir(pinDir)
	if err != nil {
		return
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		count++
	}
	XDPPinnedMapCount.Set(float64(count))
}
