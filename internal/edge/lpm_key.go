package edge

import "encoding/binary"

type IPv4Key struct {
	PrefixLen uint32
	Addr      uint32
}

type StoreID uint64

func (k IPv4Key) StoreKey() StoreID {
	return StoreID(uint64(k.PrefixLen)<<32 | uint64(k.Addr))
}

func HostKey(a, b, c, d byte) IPv4Key {
	return IPv4Key{PrefixLen: 32, Addr: beToBPFAddr(addrBE(a, b, c, d))}
}

func (k IPv4Key) BEAddr() uint32 {
	return bpfAddrToBE(k.Addr)
}

func AddrFrom4(v4 [4]byte) uint32 {
	return binary.BigEndian.Uint32(v4[:])
}

func ParseHost(s string) (uint32, bool) {
	addr, ok := parseIPv4(s)
	return addr, ok
}

func ParsePrefix(s string) (IPv4Key, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] != '/' {
			continue
		}
		addr, ok := parseIPv4(s[:i])
		if !ok {
			return IPv4Key{}, false
		}
		plen, ok := parseUint8(s[i+1:])
		if !ok || plen > 32 {
			return IPv4Key{}, false
		}
		return IPv4Key{PrefixLen: uint32(plen), Addr: beToBPFAddr(maskIPv4(addr, plen))}, true
	}
	addr, ok := parseIPv4(s)
	if !ok {
		return IPv4Key{}, false
	}
	return IPv4Key{PrefixLen: 32, Addr: beToBPFAddr(addr)}, true
}

func parseIPv4(s string) (uint32, bool) {
	if len(s) < 7 || len(s) > 15 {
		return 0, false
	}
	var (
		oct  uint32
		bits uint32
		dots int
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			oct = oct*10 + uint32(c-'0')
			if oct > 255 {
				return 0, false
			}
		case c == '.':
			if dots >= 3 || i == 0 || s[i-1] == '.' {
				return 0, false
			}
			bits = bits<<8 | oct
			oct = 0
			dots++
		default:
			return 0, false
		}
	}
	if dots != 3 {
		return 0, false
	}
	return bits<<8 | oct, true
}

func parseUint8(s string) (uint8, bool) {
	if s == "" || len(s) > 2 {
		return 0, false
	}
	var n uint8
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + uint8(c-'0')
	}
	return n, true
}

func maskIPv4(addr uint32, plen uint8) uint32 {
	if plen == 0 {
		return 0
	}
	if plen >= 32 {
		return addr
	}
	mask := uint32(0xffffffff) << (32 - plen)
	return addr & mask
}

func addrBE(a, b, c, d byte) uint32 {
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d)
}

func beToBPFAddr(be uint32) uint32 {
	var b [4]byte
	b[0] = byte(be >> 24)
	b[1] = byte(be >> 16)
	b[2] = byte(be >> 8)
	b[3] = byte(be)
	return binary.LittleEndian.Uint32(b[:])
}

func bpfAddrToBE(bpfAddr uint32) uint32 {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], bpfAddr)
	return binary.BigEndian.Uint32(b[:])
}

func ToBPFAddr(be uint32) uint32 {
	return beToBPFAddr(be)
}

func MergeHosts(dst map[uint32]struct{}, lists ...[]string) {
	for _, list := range lists {
		for _, member := range list {
			if addr, ok := ParseHost(member); ok {
				dst[beToBPFAddr(addr)] = struct{}{}
			}
		}
	}
}

func MergeDenyV4(hosts map[uint32]struct{}, prefixes map[StoreID]IPv4Key, lists ...[]string) {
	for _, list := range lists {
		for _, member := range list {
			key, ok := ParsePrefix(member)
			if !ok {
				continue
			}
			if key.PrefixLen == 32 {
				hosts[key.Addr] = struct{}{}
				continue
			}
			prefixes[key.StoreKey()] = key
		}
	}
}

func MergePrefixes(dst map[StoreID]IPv4Key, members []string) {
	for _, member := range members {
		key, ok := ParsePrefix(member)
		if !ok {
			continue
		}
		dst[key.StoreKey()] = key
	}
}
