package ingestion

import (
	"context"
	"hash/crc32"
	"strings"
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type tlsBlocklistSnapshot struct {
	blocked map[uint32]struct{}
}

type DeviceFilter struct {
	settings             *SettingsWatcher
	blockedTLS           atomic.Pointer[tlsBlocklistSnapshot]
	osFingerprintEnabled atomic.Bool
}

func NewDeviceFilter(settings *SettingsWatcher) *DeviceFilter {
	f := &DeviceFilter{settings: settings}
	f.osFingerprintEnabled.Store(true)
	f.reloadBlocklist()
	if settings != nil {
		settings.AddChangeListener(func(_ *DynamicConfig) {
			f.reloadBlocklist()
		})
	}
	return f
}

func (f *DeviceFilter) SetOSFingerprintEnabled(enabled bool) {
	f.osFingerprintEnabled.Store(enabled)
}

func (f *DeviceFilter) reloadBlocklist() {
	if f == nil {
		return
	}
	var hashes []string
	if f.settings != nil {
		hashes = parseCommaList(f.settings.Get().TLSHashBlocklist)
	}
	m := make(map[uint32]struct{}, len(hashes))
	for _, h := range hashes {
		m[crc32.ChecksumIEEE([]byte(h))] = struct{}{}
	}
	f.blockedTLS.Store(&tlsBlocklistSnapshot{blocked: m})
}

func (f *DeviceFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil {
		return nil
	}
	var blocked map[uint32]struct{}
	if snap := f.blockedTLS.Load(); snap != nil {
		blocked = snap.blocked
	}

	if evt.TLSHash != "" && len(blocked) > 0 {
		h := crc32.ChecksumIEEE([]byte(evt.TLSHash))
		if _, onList := blocked[h]; onList {
			addFraudSignal(evt, FraudReasonTLSBlocklist)
		}
	}
	if deviceHintsMismatch(evt.SecCHUA, evt.UA) {
		addFraudSignal(evt, FraudReasonDeviceMismatch)
	}
	if tlsFingerprintImpersonating(evt.UA, []byte(evt.TLSJA3), []byte(evt.TLSJA4), []byte(evt.TLSHash)) {
		addFraudSignal(evt, FraudReasonDeviceMismatch)
	}
	if f.osFingerprintEnabled.Load() && evt.UA != "" {
		if evt.TCPTTLSet == 0 {
			metrics.OSFingerprintSkippedTotal.WithLabelValues("no_tcp_headers").Inc()
		} else if osFingerprintMismatch(evt.UA, evt.TCPTTL, evt.TCPWindowSet, evt.TCPWindow) {
			metrics.OSFingerprintMismatchTotal.Inc()
			addFraudSignal(evt, FraudReasonOSFingerprint)
		}
	}
	return nil
}

func parseCommaList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func deviceHintsMismatch(secCHUA, ua string) bool {
	if secCHUA == "" {
		return false
	}
	if ua == "" {
		return true
	}
	if strings.Contains(secCHUA, "Chrome") &&
		!strings.Contains(ua, "Chrome") &&
		!strings.Contains(ua, "Chromium") {
		return true
	}
	return false
}
