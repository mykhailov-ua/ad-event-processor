//go:build linux

package licensing

import "bytes"

func trimHWIDBytes(b []byte) string {
	return string(bytes.TrimSpace(b))
}

func readHWIDFile(pathID uint8, suffix []byte) string {
	return hwidReadString(pathID, suffix)
}
