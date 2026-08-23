package edge

import (
	"encoding/binary"
	"net"
)

type IPv6Key struct {
	PrefixLen uint32
	Addr      [16]byte
}

func (k IPv6Key) StoreKey() StoreID {
	hi := binary.BigEndian.Uint64(k.Addr[0:8])
	lo := binary.BigEndian.Uint64(k.Addr[8:16])
	return StoreID(uint64(k.PrefixLen)<<32) ^ StoreID(hi) ^ StoreID(lo)
}

func ParseIPv6Host(s string) (IPv6Key, bool) {
	ip := net.ParseIP(s)
	if ip == nil {
		return IPv6Key{}, false
	}
	ip = ip.To16()
	if ip == nil || ip.To4() != nil {
		return IPv6Key{}, false
	}
	var key IPv6Key
	key.PrefixLen = 128
	copy(key.Addr[:], ip)
	return key, true
}

func ParseIPv6Prefix(s string) (IPv6Key, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] != '/' {
			continue
		}
		ip := net.ParseIP(s[:i])
		if ip == nil {
			return IPv6Key{}, false
		}
		ip = ip.To16()
		if ip == nil || ip.To4() != nil {
			return IPv6Key{}, false
		}
		plen, ok := parseUint8(s[i+1:])
		if !ok || plen > 128 {
			return IPv6Key{}, false
		}
		maskIPv6(ip, plen)
		var key IPv6Key
		key.PrefixLen = uint32(plen)
		copy(key.Addr[:], ip)
		return key, true
	}
	return ParseIPv6Host(s)
}

func maskIPv6(ip net.IP, plen uint8) {
	if plen >= 128 {
		return
	}
	byteIdx := plen / 8
	bitIdx := plen % 8
	for i := byteIdx + 1; i < 16; i++ {
		ip[i] = 0
	}
	if bitIdx > 0 {
		mask := byte(0xff << (8 - bitIdx))
		ip[byteIdx] &= mask
	} else if byteIdx < 16 {
		ip[byteIdx] = 0
	}
}

func MergeIPv6Hosts(dst map[StoreID]IPv6Key, lists ...[]string) {
	for _, list := range lists {
		for _, member := range list {
			key, ok := ParseIPv6Host(member)
			if !ok {
				continue
			}
			dst[key.StoreKey()] = key
		}
	}
}

func MergeDenyV6(hosts map[StoreID]IPv6Key, prefixes map[StoreID]IPv6Key, lists ...[]string) {
	for _, list := range lists {
		for _, member := range list {
			key, ok := ParseIPv6Prefix(member)
			if !ok {
				continue
			}
			if key.PrefixLen == 128 {
				hosts[key.StoreKey()] = key
				continue
			}
			prefixes[key.StoreKey()] = key
		}
	}
}

func MergeIPv6Prefixes(dst map[StoreID]IPv6Key, members []string) {
	for _, member := range members {
		key, ok := ParseIPv6Prefix(member)
		if !ok {
			continue
		}
		dst[key.StoreKey()] = key
	}
}
