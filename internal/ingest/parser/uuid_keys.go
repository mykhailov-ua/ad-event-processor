package parser

import (
	"unsafe"

	"github.com/google/uuid"
)

var (
	delimiterTable [256]byte
	hexLookup      [256]byte
)

func init() {
	delimiterTable[','] = 1
	delimiterTable['}'] = 1
	delimiterTable[']'] = 1
	delimiterTable[' '] = 1
	delimiterTable['\t'] = 1
	delimiterTable['\n'] = 1
	delimiterTable['\r'] = 1

	for i := range hexLookup {
		hexLookup[i] = 0xff
	}
	for i := byte('0'); i <= '9'; i++ {
		hexLookup[i] = i - '0'
	}
	for i := byte('a'); i <= 'f'; i++ {
		hexLookup[i] = i - 'a' + 10
	}
	for i := byte('A'); i <= 'F'; i++ {
		hexLookup[i] = i - 'A' + 10
	}
}

func isDelimiter(b byte) bool {
	return delimiterTable[b] != 0
}

func IsDelimiter(b byte) bool { return isDelimiter(b) }

func ParseUUID(b []byte, dst *uuid.UUID) bool {
	if len(b) == 16 {
		copy(dst[:], b)
		return true
	}
	if len(b) != 36 {
		return false
	}
	if b[8] != '-' || b[13] != '-' || b[18] != '-' || b[23] != '-' {
		return false
	}

	decode := func(h, l byte) (byte, bool) {
		vh := hexLookup[h]
		vl := hexLookup[l]
		if vh == 0xff || vl == 0xff {
			return 0, false
		}
		return (vh << 4) | vl, true
	}

	var ok bool
	dst[0], ok = decode(b[0], b[1])
	if !ok {
		return false
	}
	dst[1], ok = decode(b[2], b[3])
	if !ok {
		return false
	}
	dst[2], ok = decode(b[4], b[5])
	if !ok {
		return false
	}
	dst[3], ok = decode(b[6], b[7])
	if !ok {
		return false
	}

	dst[4], ok = decode(b[9], b[10])
	if !ok {
		return false
	}
	dst[5], ok = decode(b[11], b[12])
	if !ok {
		return false
	}

	dst[6], ok = decode(b[14], b[15])
	if !ok {
		return false
	}
	dst[7], ok = decode(b[16], b[17])
	if !ok {
		return false
	}

	dst[8], ok = decode(b[19], b[20])
	if !ok {
		return false
	}
	dst[9], ok = decode(b[21], b[22])
	if !ok {
		return false
	}

	dst[10], ok = decode(b[24], b[25])
	if !ok {
		return false
	}
	dst[11], ok = decode(b[26], b[27])
	if !ok {
		return false
	}
	dst[12], ok = decode(b[28], b[29])
	if !ok {
		return false
	}
	dst[13], ok = decode(b[30], b[31])
	if !ok {
		return false
	}
	dst[14], ok = decode(b[32], b[33])
	if !ok {
		return false
	}
	dst[15], ok = decode(b[34], b[35])
	return ok
}

type KeyID uint8

const (
	KeyUnknown KeyID = iota
	KeyType
	KeyUserID
	KeyPayload
	KeyClickID
	KeyCampaignID
	KeyPlacementID
	KeyFBCLID
	KeyGCLID
	KeyTTCLID
	KeyMSCLKID
	KeyTBLCI
	KeyOBClickID
	KeyEventID
	KeyTxID
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

var whitespaceTable [256]byte

func init() {
	whitespaceTable[' '] = 1
	whitespaceTable['\t'] = 1
	whitespaceTable['\n'] = 1
	whitespaceTable['\r'] = 1
}

func loadU32(b []byte) uint32 {
	return *(*uint32)(unsafe.Pointer(&b[0]))
}

func LoadU32(b []byte) uint32 { return loadU32(b) }

func loadU64(b []byte) uint64 {
	return *(*uint64)(unsafe.Pointer(&b[0]))
}

func LoadU64(b []byte) uint64 { return loadU64(b) }

func MatchTrackKey(key []byte) KeyID {
	switch len(key) {
	case 4:
		if loadU32(key) == u32Type {
			return KeyType
		}
	case 7:
		switch loadU32(key) {
		case u32Payl:
			if key[4] == 'o' && key[5] == 'a' && key[6] == 'd' {
				return KeyPayload
			}
		case u32User:
			if key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
				return KeyUserID
			}
		case 0x6c63736d:
			if key[4] == 'k' && key[5] == 'i' && key[6] == 'd' {
				return KeyMSCLKID
			}
		}
	case 8:
		switch loadU64(key) {
		case u64ClickID:
			return KeyClickID
		case u64EventID:
			return KeyEventID
		}
	case 11:
		if loadU64(key) == u64OBClick && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return KeyOBClickID
		}
		if loadU64(key) == u64Campaign && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return KeyCampaignID
		}
	case 12:
		if loadU64(key) == u64Placement && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return KeyPlacementID
		}
	case 5:
		if loadU32(key) == u32TxID && key[4] == 'd' {
			return KeyTxID
		}
		if loadU32(key) == 0x696c6367 && key[4] == 'd' {
			return KeyGCLID
		}
		if loadU32(key) == 0x636c6274 && key[4] == 'i' {
			return KeyTBLCI
		}
	case 6:
		switch loadU32(key) {
		case 0x6c636266:
			if key[4] == 'i' && key[5] == 'd' {
				return KeyFBCLID
			}
		case 0x6c637474:
			if key[4] == 'i' && key[5] == 'd' {
				return KeyTTCLID
			}
		}
	}
	return KeyUnknown
}
