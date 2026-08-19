//go:build linux && amd64

package licensing

import (
	"syscall"
)

//go:noescape
func hwidRawOpenRead(pathPtr *byte, pathLen int, bufPtr *byte, bufLen int) int64

func hwidRead(pathID uint8, suffix, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, syscall.EINVAL
	}
	var pathBuf [256]byte
	pathLen := decodeHWIDPath(pathID, suffix, pathBuf[:])
	if pathLen <= 0 {
		return 0, syscall.EINVAL
	}
	n := hwidRawOpenRead(&pathBuf[0], pathLen, &buf[0], len(buf))
	if n < 0 {
		return 0, syscall.Errno(-n)
	}
	return int(n), nil
}

func hwidReadString(pathID uint8, suffix []byte) string {
	var buf [512]byte
	n, err := hwidRead(pathID, suffix, buf[:])
	if err != nil || n <= 0 {
		return ""
	}
	return trimHWIDBytes(buf[:n])
}

func hwidReadStringFromIDs(pathID uint8, midSuffix []byte, tailID uint8) string {
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
	var buf [512]byte
	ret := hwidRawOpenRead(&pathBuf[0], fullLen, &buf[0], len(buf))
	if ret < 0 {
		return ""
	}
	return trimHWIDBytes(buf[:ret])
}
