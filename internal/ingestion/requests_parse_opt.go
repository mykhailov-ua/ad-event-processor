package ingestion

import (
	"unsafe"
)

type keyID uint8

const (
	keyUnknown keyID = iota
	keyType
	keyUserID
	keyPayload
	keyClickID
	keyCampaignID
	keyPlacementID
	keyFBCLID
	keyGCLID
	keyTTCLID
	keyMSCLKID
	keyTBLCI
	keyOBClickID
	keyEventID
	keyTxID
)

const (
	u32Type      uint32 = 0x65707974
	u32Payl      uint32 = 0x6c796170
	u32User      uint32 = 0x72657375
	u64ClickID   uint64 = 0x64695f6b63696c63
	u64EventID   uint64 = 0x64695f746e657665
	u64Campaign  uint64 = 0x6e676961706d6163
	u64Placement uint64 = 0x6e65636d65636170
	u64OBClick   uint64 = 0x6b63696c635f626f
	u32TxID      uint32 = 0x695f7874
)

var jsonWhitespace [256]byte

func init() {
	jsonWhitespace[' '] = 1
	jsonWhitespace['\t'] = 1
	jsonWhitespace['\n'] = 1
	jsonWhitespace['\r'] = 1
}

func loadU32(b []byte) uint32 {
	return *(*uint32)(unsafe.Pointer(&b[0]))
}

func loadU64(b []byte) uint64 {
	return *(*uint64)(unsafe.Pointer(&b[0]))
}

func matchTrackKey(key []byte) keyID {
	switch len(key) {
	case 4:
		if loadU32(key) == u32Type {
			return keyType
		}
	case 7:
		switch loadU32(key) {
		case u32Payl:
			if key[4] == 'o' && key[5] == 'a' && key[6] == 'd' {
				return keyPayload
			}
		case u32User:
			if key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
				return keyUserID
			}
		case 0x6c63736d:
			if key[4] == 'k' && key[5] == 'i' && key[6] == 'd' {
				return keyMSCLKID
			}
		}
	case 8:
		switch loadU64(key) {
		case u64ClickID:
			return keyClickID
		case u64EventID:
			return keyEventID
		}
	case 11:
		if loadU64(key) == u64OBClick && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return keyOBClickID
		}
		if loadU64(key) == u64Campaign && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return keyCampaignID
		}
	case 12:
		if loadU64(key) == u64Placement && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return keyPlacementID
		}
	case 5:
		if loadU32(key) == u32TxID && key[4] == 'd' {
			return keyTxID
		}
		if loadU32(key) == 0x696c6367 && key[4] == 'd' {
			return keyGCLID
		}
		if loadU32(key) == 0x636c6274 && key[4] == 'i' {
			return keyTBLCI
		}
	case 6:
		switch loadU32(key) {
		case 0x6c636266:
			if key[4] == 'i' && key[5] == 'd' {
				return keyFBCLID
			}
		case 0x6c637474:
			if key[4] == 'i' && key[5] == 'd' {
				return keyTTCLID
			}
		}
	}
	return keyUnknown
}

func assignTrackStringField(v *TrackRequest, kid keyID, valBytes []byte) {
	switch kid {
	case keyType:
		v.Type = unsafeString(valBytes)
	case keyUserID:
		v.UserID = unsafeString(valBytes)
	case keyClickID:
		v.ClickID = unsafeString(valBytes)
	case keyPlacementID:
		v.PlacementID = unsafeString(valBytes)
	case keyFBCLID:
		v.fbclid = unsafeString(valBytes)
	case keyGCLID:
		v.gclid = unsafeString(valBytes)
	case keyTTCLID:
		v.ttclid = unsafeString(valBytes)
	case keyMSCLKID:
		v.msclkid = unsafeString(valBytes)
	case keyTBLCI:
		v.tblci = unsafeString(valBytes)
	case keyOBClickID:
		v.obClickID = unsafeString(valBytes)
	case keyEventID:
		v.eventID = unsafeString(valBytes)
	case keyTxID:
		v.txID = unsafeString(valBytes)
	}
}

func ParseTrackRequestJSONOpt(v *TrackRequest, data []byte) error {
	return parseTrackRequestJSON(v, data)
}

func parseTrackRequestJSON(v *TrackRequest, data []byte) error {
	v.resetForParse()
	if len(data) == 0 {
		return errMalformedJSON
	}
	_ = data[len(data)-1]

	n := len(data)
	bud := newJSONScanBudget()
	i, ok := skipJSONWSBudget(data, 0, n, &bud)
	if !ok || i >= n || data[i] != '{' {
		return errMalformedJSON
	}
	i++

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return errMalformedJSON
		}
		if data[i] == '}' {
			return nil
		}
		if data[i] != '"' {
			return errMalformedJSON
		}
		i++

		keyStart := i
		for i < n && data[i] != '"' {
			if data[i] == '\\' {
				return errMalformedJSON
			}
			i++
		}
		if i >= n {
			return errMalformedJSON
		}
		keyEnd := i
		i++
		if !jsonTrackKeyOK(data[keyStart:keyEnd]) {
			return errMalformedJSON
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n || data[i] != ':' {
			return errMalformedJSON
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return errMalformedJSON
		}

		kid := matchTrackKey(data[keyStart:keyEnd])
		keyBytes := data[keyStart:keyEnd]
		switch kid {
		case keyType, keyUserID, keyClickID, keyPlacementID, keyFBCLID, keyGCLID, keyTTCLID, keyMSCLKID, keyTBLCI, keyOBClickID, keyEventID, keyTxID:
			if data[i] != '"' {
				return errMalformedJSON
			}
			valStart := i + 1
			end, ok := scanJSONStringEnd(data, i, n, &bud)
			if !ok {
				return errMalformedJSON
			}
			valBytes := data[valStart : end-1]
			i = end

			assignTrackStringField(v, kid, valBytes)
		case keyCampaignID:
			if data[i] != '"' {
				return errMalformedJSON
			}
			valStart := i + 1
			end, ok := scanJSONLiteralStringEnd(data, i, n, &bud)
			if !ok {
				return errMalformedJSON
			}
			if !ParseUUID(data[valStart:end-1], &v.CampaignID) {
				return errMalformedJSON
			}
			i = end
		case keyPayload:
			valStart := i
			valEnd, err := skipJSONValueBudget(data, i, &bud)
			if err != nil {
				return err
			}
			v.Payload = data[valStart:valEnd]
			i = valEnd
		default:
			if idx, ok := subKeyIndex(keyBytes); ok {
				if data[i] != '"' {
					return errMalformedJSON
				}
				valStart := i + 1
				end, ok := scanJSONStringEnd(data, i, n, &bud)
				if !ok {
					return errMalformedJSON
				}
				v.subs[idx-1] = unsafeString(data[valStart : end-1])
				i = end
			} else {
				valEnd, err := skipJSONValueBudget(data, i, &bud)
				if err != nil {
					return err
				}
				i = valEnd
			}
		}

		if !bud.consumeKeyPair() {
			return errMalformedJSON
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return errMalformedJSON
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return nil
		default:
			return errMalformedJSON
		}
	}

	return errMalformedJSON
}
