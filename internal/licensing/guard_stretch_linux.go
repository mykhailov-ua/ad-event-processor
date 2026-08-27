//go:build linux && license_guard

package licensing

import (
	"encoding/binary"
	"time"

	"ad-event-processor/internal/config"

	"golang.org/x/crypto/argon2"
)

const guardTamperStretchTimeout = 500 * time.Millisecond

var guardTamperStretchHook func(reason string, textHash [32]byte)

func runTamperStretch(reason string, textHash [32]byte) {
	if guardTamperStretchHook != nil {
		guardTamperStretchHook(reason, textHash)
		return
	}
	if !config.LicenseGuardStretchEnabled() {
		return
	}
	ikm := make([]byte, 4)
	binary.LittleEndian.PutUint32(ikm, GuardTextFingerprint(textHash))
	salt := []byte(reason)
	done := make(chan struct{})
	go func() {
		_ = argon2.IDKey(ikm, salt, 1, hwidArgonMemory, hwidArgonThreads, hwidArgonKeyLen)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(guardTamperStretchTimeout):
	}
}

func SetGuardTamperStretchHookForTest(fn func(string, [32]byte)) func() {
	prev := guardTamperStretchHook
	guardTamperStretchHook = fn
	return func() { guardTamperStretchHook = prev }
}
