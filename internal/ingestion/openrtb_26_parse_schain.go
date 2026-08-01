package ingestion

import "bytes"

//go:noinline
func parseSchainNodesAt(payload []byte, schainAt int) SchainNodes {
	var out SchainNodes
	if schainAt < 0 || schainAt >= len(payload) {
		return out
	}
	n := len(payload)
	nodesAt := bytes.Index(payload[schainAt:], openrtbKeyNodes)
	if nodesAt < 0 {
		return out
	}
	i := schainAt + nodesAt + len(openrtbKeyNodes)
	for i < n && payload[i] != '[' {
		i++
	}
	if i >= n {
		return out
	}
	i++
	for i < n && out.Count < schainNodeMax {
		if payload[i] == ']' {
			break
		}
		if payload[i] != '{' {
			i++
			continue
		}
		objEnd := i
		depth := 0
		for objEnd < n {
			if payload[objEnd] == '{' {
				depth++
			} else if payload[objEnd] == '}' {
				depth--
				if depth == 0 {
					break
				}
			}
			objEnd++
		}
		if objEnd >= n {
			break
		}
		obj := payload[i : objEnd+1]
		node := SchainNode{}
		if asiRel := bytes.Index(obj, openrtbKeyAsi); asiRel >= 0 {
			node.ASILen = uint8(parseQuotedField(obj, asiRel+len(openrtbKeyAsi), node.ASI[:]))
		}
		if sidRel := bytes.Index(obj, openrtbKeySid); sidRel >= 0 {
			node.SIDLen = uint8(parseQuotedField(obj, sidRel+len(openrtbKeySid), node.SID[:]))
		}
		out.Nodes[out.Count] = node
		out.Count++
		i = objEnd + 1
	}
	return out
}
