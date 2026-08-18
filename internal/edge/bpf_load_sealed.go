package edge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/cilium/ebpf"
)

const sealedEdgeAssetLabel = "edge-bpf"

func sealedEdgeBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "edge", "edge_sealed.bin")
}

func sealedEdgeBlob() ([]byte, error) {
	path := sealedEdgeBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func loadEdgeObjectsFromSealed(objs *EdgeObjects, opts *ebpf.CollectionOptions) error {
	if config.LicenseAssetsUnsealed() {
		return os.ErrNotExist
	}
	sealed, err := sealedEdgeBlob()
	if err != nil {
		return err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.EdgeBPFSealFailTotal.Inc()
		return fmt.Errorf("sealed bpf mck: %w", err)
	}
	elf, err := licensing.OpenAsset(sealedEdgeAssetLabel, sealed, mck)
	if err != nil {
		metrics.EdgeBPFSealFailTotal.Inc()
		return fmt.Errorf("sealed bpf open: %w", err)
	}
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(elf))
	if err != nil {
		metrics.EdgeBPFSealFailTotal.Inc()
		return fmt.Errorf("sealed bpf spec: %w", err)
	}
	return populateEdgeObjectsFromSpec(objs, spec, opts)
}

func edgePlaintextELF() ([]byte, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("edge bpf elf: caller")
	}
	candidates := []string{
		filepath.Join(filepath.Dir(file), "edge_bpfel.o"),
		filepath.Join("internal", "edge", "bpf", "edge_bpfel.o"),
	}
	var lastErr error
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil && len(data) > 0 {
			return data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = os.ErrNotExist
	}
	return nil, lastErr
}

func populateEdgeObjectsFromSpec(objs *EdgeObjects, spec *ebpf.CollectionSpec, opts *ebpf.CollectionOptions) error {
	if objs == nil || spec == nil {
		return fmt.Errorf("nil edge bpf load target")
	}
	var collOpts ebpf.CollectionOptions
	if opts != nil {
		collOpts = *opts
	}
	coll, err := ebpf.NewCollectionWithOptions(spec, collOpts)
	if err != nil {
		delete(spec.Programs, EdgeProgXdpSynCookie)
		coll, err = ebpf.NewCollectionWithOptions(spec, collOpts)
		if err != nil {
			return err
		}
	}
	prog := coll.Programs[EdgeProgXdpEdgeFilter]
	if prog == nil {
		coll.Close()
		return fmt.Errorf("missing program %s", EdgeProgXdpEdgeFilter)
	}
	objs.XdpEdgeFilter = prog
	assignMap := func(name string, dst **ebpf.Map) {
		if m := coll.Maps[name]; m != nil {
			*dst = m
		}
	}
	assignMap(EdgeMapAllowV4, &objs.AllowV4)
	assignMap(EdgeMapBlocklistV4, &objs.BlocklistV4)
	assignMap(EdgeMapConfig, &objs.Config)
	assignMap(EdgeMapGlobalSyn, &objs.GlobalSyn)
	assignMap(EdgeMapProgArray, &objs.ProgArray)
	assignMap(EdgeMapRatelimitV4, &objs.RatelimitV4)
	assignMap(EdgeMapRstRatelimitV4, &objs.RstRatelimitV4)
	assignMap(EdgeMapStats, &objs.Stats)
	assignMap(EdgeMapSynRatelimitV4, &objs.SynRatelimitV4)
	assignMap(EdgeMapSynSubnetRatelimitV4, &objs.SynSubnetRatelimitV4)
	assignMap(EdgeMapViolations, &objs.Violations)
	assignMap(EdgeMapFingerprints, &objs.Fingerprints)
	for name := range coll.Programs {
		delete(coll.Programs, name)
	}
	for name := range coll.Maps {
		delete(coll.Maps, name)
	}
	coll.Close()
	return nil
}
