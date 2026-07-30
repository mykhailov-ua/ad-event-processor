package bpf

import (
	"testing"
)

func FuzzDecodeFingerprint(f *testing.F) {
	seed := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x0a, 0x00, 0x00, 0x01,
		0xde, 0xad, 0xbe, 0xef,
		0x01, 0x02,
		0x40,
		0xb4,
	}
	f.Add(seed)
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic in decodeFingerprint: %v", r)
			}
		}()
		if len(data) < 20 {
			return
		}
		_ = decodeFingerprint(data)
	})
}

func FuzzConfigUpdate(f *testing.F) {
	f.Add(true, false)
	f.Fuzz(func(t *testing.T, cookie, disableFingerprint bool) {
		objs := loadTestObjects(t)
		opts := InitOptions{
			SynCookieEnabled:   cookie,
			DisableFingerprint: disableFingerprint,
		}
		_ = InitConfigWith(objs.Config, opts)
	})
}

func FuzzStatsAggregation(f *testing.F) {
	f.Add(true)
	f.Fuzz(func(t *testing.T, isNil bool) {
		if isNil {
			_, _ = AggregateStats(nil)
		} else {
			objs := loadTestObjects(t)
			_, _ = AggregateStats(objs.Stats)
		}
	})
}

func FuzzDecodeViolation(f *testing.F) {
	seed := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x0a, 0x00, 0x00, 0x02,
		0x01,
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic in decodeViolation: %v", r)
			}
		}()
		if len(data) < 13 {
			return
		}
		_ = decodeViolation(data)
	})
}
