package ingestion

//go:noinline
func parseSchainNodesAt(payload []byte, schainAt int) SchainNodes {
	var out SchainNodes
	if schainAt < 0 || schainAt >= len(payload) {
		return out
	}
	win := payload[schainAt:]
	sw := scanSchainWindow(win)
	if sw.idxNodes < 0 {
		return out
	}
	n := len(payload)
	i := schainAt + sw.idxNodes + len(openrtbKeyNodes)
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
		sn := scanSchainNodeObject(obj)
		node := SchainNode{}
		if sn.idxAsi >= 0 {
			node.ASILen = uint8(parseQuotedField(obj, sn.idxAsi+len(openrtbKeyAsi), node.ASI[:]))
		}
		if sn.idxSid >= 0 {
			node.SIDLen = uint8(parseQuotedField(obj, sn.idxSid+len(openrtbKeySid), node.SID[:]))
		}
		out.Nodes[out.Count] = node
		out.Count++
		i = objEnd + 1
	}
	return out
}
