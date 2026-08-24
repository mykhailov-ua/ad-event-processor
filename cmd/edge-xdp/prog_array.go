package main

import (
	"fmt"

	"ad-event-processor/internal/edge"

	"github.com/cilium/ebpf"
)

func wireProgArray(objs *edge.EdgeObjects) error {
	if objs.ProgArray == nil {
		return fmt.Errorf("prog_array not loaded")
	}
	if objs.XdpSynCookie == nil {
		return nil
	}
	key := uint32(0)
	return objs.ProgArray.Update(&key, objs.XdpSynCookie, ebpf.UpdateAny)
}
