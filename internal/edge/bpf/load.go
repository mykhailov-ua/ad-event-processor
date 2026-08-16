package bpf

import (
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/cilium/ebpf"
)

// LoadEdgeObjectsLenient loads edge BPF objects. When the optional xdp_syn_cookie
// program fails verifier load (common on some kernels / lab hosts), it retries
// without that program so blocklist attach can still proceed with syn_cookie disabled.
// When a sealed blob is present and license mode requires sealing, decrypts with MCK first.
func LoadEdgeObjectsLenient(objs *EdgeObjects, opts *ebpf.CollectionOptions) error {
	_, sealedErr := sealedEdgeBlob()
	if sealedErr == nil && !config.LicenseAssetsUnsealed() {
		if err := loadEdgeObjectsFromSealed(objs, opts); err != nil {
			return err
		}
		return nil
	}
	if err := LoadEdgeObjects(objs, opts); err == nil {
		return nil
	}

	spec, err := LoadEdge()
	if err != nil {
		return err
	}
	delete(spec.Programs, EdgeProgXdpSynCookie)

	var collOpts ebpf.CollectionOptions
	if opts != nil {
		collOpts = *opts
	}
	coll, err := ebpf.NewCollectionWithOptions(spec, collOpts)
	if err != nil {
		return err
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
