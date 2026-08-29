package openrtb

const (
	SchainNodeMax     = 8
	schainNodeMax     = SchainNodeMax
	schainASIMax      = 64
	schainSIDMax      = 64
	schainAllowlistAS = 32
)

type SupplyChainNode struct {
	ASI    [schainASIMax]byte
	ASILen uint8
	SID    [schainSIDMax]byte
	SIDLen uint8
}

type SupplyChainNodes struct {
	Count uint8
	Nodes [schainNodeMax]SupplyChainNode
}

type SupplyChainAllowlistSnapshot struct {
	Allowed map[string]struct{}
}

func SchainAllowKey(asi, sid []byte) string {
	if len(asi) == 0 || len(sid) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(asi)+1+len(sid))
	buf = append(buf, asi...)
	buf = append(buf, '|')
	buf = append(buf, sid...)
	return string(buf)
}

func ValidateSupplyChainNodes(nodes SupplyChainNodes, allow *SupplyChainAllowlistSnapshot) bool {
	if nodes.Count == 0 {
		return true
	}
	if allow == nil || len(allow.Allowed) == 0 {
		return true
	}
	for i := uint8(0); i < nodes.Count; i++ {
		n := nodes.Nodes[i]
		if n.ASILen == 0 || n.SIDLen == 0 {
			continue
		}
		key := SchainAllowKey(n.ASI[:n.ASILen], n.SID[:n.SIDLen])
		if _, ok := allow.Allowed[key]; !ok {
			return false
		}
	}
	return true
}
