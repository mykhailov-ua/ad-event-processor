package edge

import (
	"os"

	"github.com/cilium/ebpf"
)

const (
	DefaultSynLimit       = 64
	DefaultPPSRate        = 2000
	DefaultGlobalSynLimit = 50000
	DefaultAssumedCPUs    = 8
)

type InitOptions struct {
	SynCookieEnabled   bool
	DisableFingerprint bool
}

func DefaultConfig(opts InitOptions) EdgeEdgeConfig {
	cfg := EdgeEdgeConfig{
		SynLimit:           DefaultSynLimit,
		PpsRate:            DefaultPPSRate,
		GlobalSynLimit:     DefaultGlobalSynLimit,
		AssumedCpus:        DefaultAssumedCPUs,
		SynSubnetLimit:     DefaultSynSubnetLimit,
		SynCookieEnabled:   0,
		FingerprintEnabled: 1,
	}
	if opts.SynCookieEnabled {
		cfg.SynCookieEnabled = 1
	}
	if opts.DisableFingerprint {
		cfg.FingerprintEnabled = 0
	}
	return cfg
}

func InitConfigFromEnv(m *ebpf.Map) error {
	opts := InitOptions{}
	if v := os.Getenv("XDP_SYN_COOKIE"); v == "1" || v == "true" {
		opts.SynCookieEnabled = true
	}
	if v := os.Getenv("XDP_FINGERPRINT"); v == "0" || v == "false" {
		opts.DisableFingerprint = true
	}
	return InitConfigWith(m, opts)
}

func InitConfig(m *ebpf.Map) error {
	return InitConfigWith(m, InitOptions{})
}

func InitConfigWith(m *ebpf.Map, opts InitOptions) error {
	if m == nil {
		return nil
	}
	key := uint32(0)
	cfg := DefaultConfig(opts)
	return m.Update(&key, &cfg, ebpf.UpdateAny)
}

func SynCookieEnabled() bool {
	v := os.Getenv("XDP_SYN_COOKIE")
	return v == "1" || v == "true"
}
