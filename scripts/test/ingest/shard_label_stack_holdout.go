package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"
)

const stackArenaLo = uintptr(0xc000000000)

func appendInt64(dst []byte, v int64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	if v < 0 {
		dst = append(dst, '-')
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, buf[i:]...)
}

func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func stackUnsafeLabel(shard int) string {
	var scratch [8]byte
	return unsafeString(appendInt64(scratch[:0], int64(shard)))
}

func main() {
	const shard = 7
	want := strconv.Itoa(shard)

	safe := strconv.Itoa(shard)
	if safe != want {
		fmt.Fprintf(os.Stderr, "FAIL: strconv mismatch: got %q want %q\n", safe, want)
		os.Exit(1)
	}
	safePtr := uintptr(unsafe.Pointer(unsafe.StringData(safe)))
	if safePtr >= stackArenaLo {
		fmt.Fprintf(os.Stderr, "FAIL: strconv single-digit label in stack arena %#x\n", safePtr)
		os.Exit(1)
	}

	buggy := stackUnsafeLabel(shard)
	if buggy != want {
		fmt.Fprintf(os.Stderr, "FAIL: stack unsafe mismatch: got %q want %q\n", buggy, want)
		os.Exit(1)
	}
	buggyPtr := uintptr(unsafe.Pointer(unsafe.StringData(buggy)))
	if buggyPtr < stackArenaLo {
		fmt.Fprintf(os.Stderr, "FAIL: holdout expected stack-arena backing for unsafe pattern, got %#x\n", buggyPtr)
		os.Exit(1)
	}
	if unsafe.StringData(safe) == unsafe.StringData(buggy) {
		fmt.Fprintf(os.Stderr, "FAIL: safe and buggy share backing\n")
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "PASS: strconv.Itoa single-digit uses non-stack backing; stack unsafeString uses stack arena (P0-1 holdout)\n")
}
