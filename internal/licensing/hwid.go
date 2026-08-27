package licensing

import (
	"encoding/hex"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	hwidArgonTime    uint32 = 3
	hwidArgonMemory  uint32 = 65536 // KiB per Argon2 (64 MiB)
	hwidArgonThreads uint8  = 4
	hwidArgonKeyLen  uint32 = 32
)

type HWIDTelemetry struct {
	DMIUUID   string
	DiskID    string
	MAC       string
	CPUModel  string
	CPUCores  int
	MachineID string
}

var (
	hwidCollectFn = collectHWIDTelemetry
	hostHWIDOnce  sync.Once
	hostHWIDVal   string
)

func SetHWIDCollectForTest(fn func() HWIDTelemetry) func() {
	prev := hwidCollectFn
	if fn == nil {
		hwidCollectFn = collectHWIDTelemetry
	} else {
		hwidCollectFn = fn
	}
	resetHostHWIDCacheForTest()
	return func() {
		hwidCollectFn = prev
		resetHostHWIDCacheForTest()
	}
}

func HostHWID() string {
	hostHWIDOnce.Do(func() {
		hostHWIDVal = HashHWIDFromTelemetry(hwidCollectFn())
	})
	return hostHWIDVal
}

func resetHostHWIDCacheForTest() {
	hostHWIDOnce = sync.Once{}
	hostHWIDVal = ""
}

func HashHWIDFromTelemetry(t HWIDTelemetry) string {
	sum := argon2.IDKey(canonicalHWIDInput(t), hwidSalt(), hwidArgonTime, hwidArgonMemory, hwidArgonThreads, hwidArgonKeyLen)
	return hex.EncodeToString(sum)
}

func HWIDArgonTime() uint32      { return hwidArgonTime }
func HWIDArgonMemoryKiB() uint32 { return hwidArgonMemory }
func HWIDArgonThreads() uint8    { return hwidArgonThreads }
func HWIDArgonKeyLen() uint32    { return hwidArgonKeyLen }

func LabCollectHWID() (HWIDTelemetry, string) {
	tel := hwidCollectFn()
	return tel, HashHWIDFromTelemetry(tel)
}

func canonicalHWIDInput(t HWIDTelemetry) []byte {
	fields := []string{
		strings.TrimSpace(t.DMIUUID),
		strings.TrimSpace(t.DiskID),
		strings.TrimSpace(t.MAC),
		strings.TrimSpace(t.CPUModel),
		strconv.Itoa(t.CPUCores),
	}
	if HWIDV3Enabled() {
		fields = append(fields, strings.TrimSpace(t.MachineID))
	}
	return []byte(strings.Join(fields, "\x00"))
}

func hwidSalt() []byte {
	enc := []byte{
		0xf2, 0x9b, 0x44, 0xc1, 0x6e, 0x28, 0x91, 0x3d,
		0x57, 0xa0, 0x12, 0x8f, 0xde, 0x63, 0xb5, 0x07,
		0x4a, 0xcc, 0x39, 0x71, 0x25, 0x9e, 0x50, 0x86,
		0x1b, 0xd4, 0x6a, 0xbf, 0x03, 0x78, 0xe2, 0x5c,
	}
	salt := make([]byte, len(enc))
	for i, b := range enc {
		salt[i] = b ^ 0xa7
	}
	return salt
}
