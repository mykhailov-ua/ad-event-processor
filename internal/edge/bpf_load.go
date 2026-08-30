package edge

import (
	"fmt"

	"ad-event-processor/internal/config"

	"github.com/cilium/ebpf"
)

// LoadEdgeObjectsLenient: sealed blob when licensed, else bpf2go objects; final fallback
// loads edge_filter only (drops syn_cookie program) so dev hosts without cookie object still attach XDP.
//
// Verify:
// go test ./internal/edge/ -short -run TestLoadEdge -count=1
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
	assignMap(EdgeMapAllowV6, &objs.AllowV6)
	assignMap(EdgeMapBlocklistV4, &objs.BlocklistV4)
	assignMap(EdgeMapBlocklistV6, &objs.BlocklistV6)
	assignMap(EdgeMapBlocklistHostV4, &objs.BlocklistHostV4)
	assignMap(EdgeMapBlocklistHostV6, &objs.BlocklistHostV6)
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
