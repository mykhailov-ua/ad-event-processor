package ingestion

// appendAttributionPayload writes sub slots and ad-network click ids into evt.Payload JSON.
func appendAttributionPayload(dst, payload []byte, subs SubIDSlots, fbclid, gclid, ttclid string) []byte {
	dst = dst[:0]
	if len(payload) > 0 && payload[0] == '{' {
		if len(payload) > 1 && payload[len(payload)-1] == '}' {
			dst = append(dst, payload[:len(payload)-1]...)
		} else {
			dst = append(dst, payload...)
		}
	} else if len(payload) > 0 {
		dst = append(dst, '{')
		dst = appendJSONKeyString(dst, "payload", payload)
	} else {
		dst = append(dst, '{')
	}

	for i := range MaxSubIDs {
		if subs[i] == "" {
			continue
		}
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = appendSubKey(dst, i+1)
		dst = append(dst, ':')
		dst = appendJSONString(dst, UnsafeBytes(subs[i]))
	}
	if fbclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"fbclid":`...)
		dst = appendJSONString(dst, UnsafeBytes(fbclid))
	}
	if gclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"gclid":`...)
		dst = appendJSONString(dst, UnsafeBytes(gclid))
	}
	if ttclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"ttclid":`...)
		dst = appendJSONString(dst, UnsafeBytes(ttclid))
	}
	if len(dst) == 1 {
		return nil
	}
	dst = append(dst, '}')
	return dst
}

func appendSubKey(dst []byte, idx int) []byte {
	dst = append(dst, '"', 's', 'u', 'b')
	if idx < 10 {
		return append(dst, byte('0'+idx), '"')
	}
	return append(dst, byte('0'+idx/10), byte('0'+idx%10), '"')
}

func appendAttributionPassthrough(dst []byte, fbclid, gclid, ttclid string) []byte {
	if fbclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "fbclid="...)
		dst = append(dst, fbclid...)
	}
	if gclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "gclid="...)
		dst = append(dst, gclid...)
	}
	if ttclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "ttclid="...)
		dst = append(dst, ttclid...)
	}
	return dst
}

func appendJSONKeyString(dst []byte, key string, val []byte) []byte {
	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"', ':')
	return appendJSONString(dst, val)
}
