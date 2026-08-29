//go:build linux

package verify

import "unsafe"

func hwidDecodedPath(pathID uint8, suffix []byte) string {
	var buf [256]byte
	n := decodeHWIDPath(pathID, suffix, buf[:])
	if n <= 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(buf[:n]), n)
}

func hwidDecodedPathFromIDs(pathID uint8, midSuffix []byte, tailID uint8) string {
	var pathBuf [256]byte
	n := decodeHWIDPath(pathID, midSuffix, pathBuf[:])
	if n <= 0 {
		return ""
	}
	tail := hwidPathEnc[tailID]
	if len(tail)+n > len(pathBuf) {
		return ""
	}
	for i, b := range tail {
		pathBuf[n+i] = b ^ hwidPathXORMask
	}
	fullLen := n + len(tail)
	return unsafe.String(unsafe.SliceData(pathBuf[:fullLen]), fullLen)
}
