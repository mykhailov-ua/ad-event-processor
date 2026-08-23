//go:build linux && !amd64

package licensing

import (
	"syscall"
)

func hwidRead(pathID uint8, suffix []byte, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, syscall.EINVAL
	}
	var pathBuf [256]byte
	pathLen := decodeHWIDPath(pathID, suffix, pathBuf[:])
	if pathLen <= 0 {
		return 0, syscall.EINVAL
	}
	fd, err := syscall.Open(string(pathBuf[:pathLen]), syscall.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer syscall.Close(fd)
	return syscall.Read(fd, buf)
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
	fd, err := syscall.Open(string(pathBuf[:fullLen]), syscall.O_RDONLY, 0)
	if err != nil {
		return ""
	}
	defer syscall.Close(fd)
	var buf [512]byte
	got, err := syscall.Read(fd, buf[:])
	if err != nil || got <= 0 {
		return ""
	}
	return trimHWIDBytes(buf[:got])
}
