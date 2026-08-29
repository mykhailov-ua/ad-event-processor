package httpingress

import (
	"ad-event-processor/internal/filter"
)

func http1AssignHeader(req *Request, key, val []byte, hFlags *uint8, clValue *int) error {
	kl := len(key)
	if kl == 2 && httpFold[key[0]] == 't' && httpFold[key[1]] == 'e' {
		return http1AssignTransferEncoding(hFlags, val)
	}
	if kl == 4 {
		if httpFold[key[0]] == 'h' && httpFold[key[1]] == 'o' && httpFold[key[2]] == 's' && httpFold[key[3]] == 't' {
			req.Host = val
		}
		return nil
	}
	if kl < 6 {
		return nil
	}
	_ = key[kl-1]

	switch kl {
	case 6:
		switch foldKeyU32(key, 0) {
		case 0x65636361:
			if httpFold[key[4]] == 'p' && httpFold[key[5]] == 't' {
				req.Accept = val
			}
		case 0x6b6f6f63:
			if httpFold[key[4]] == 'i' && httpFold[key[5]] == 'e' {
				req.Cookie = val
			}
		case 0x6769726f:
			if httpFold[key[4]] == 'i' && httpFold[key[5]] == 'n' {
				req.Origin = val
			}
		}
	case 9:
		switch foldKeyU32(key, 0) {
		case 0x65722d78:
			if httpFold[key[4]] == 'a' && httpFold[key[5]] == 'l' && httpFold[key[6]] == '-' &&
				httpFold[key[7]] == 'i' && httpFold[key[8]] == 'p' {
				if len(req.ClientIP) == 0 {
					req.ClientIP = val
				}
			}
		case 0x2d636573:
			if foldKeyU32(key, 4) == 0x752d6863 && httpFold[key[8]] == 'a' {
				req.SecCHUA = val
			}
		case 0x6c742d78:
			if foldKeyU32(key, 4) == 0x616a2d73 {
				switch httpFold[key[8]] {
				case '3':
					req.TLSJA3 = val
				case '4':
					req.TLSJA4 = val
				}
			}
		case 0x63742d78:
			if foldKeyU32(key, 4) == 0x736d2d70 && httpFold[key[8]] == 's' {
				if mss, ok := parseTCPMSSHeader(val); ok {
					req.TCPMSS = mss
					req.TCPMSSSet = 1
				}
			} else if foldKeyU32(key, 4) == 0x74742d70 && httpFold[key[8]] == 'l' {
				if ttl, ok := parseTCPTTLHeader(val); ok {
					req.TCPTTL = ttl
					req.TCPTTLSet = 1
				}
			} else if foldKeyU32(key, 4) == 0x69732d70 && httpFold[key[8]] == 'g' {
				if sig, ok := filter.ParseTCPSigHeader(val); ok {
					req.TCPSig = sig
					req.TCPSigSet = 1
				}
			}
		}
	case 10:
		switch foldKeyU32(key, 0) {
		case 0x72657375:
			if foldKeyU32(key, 4) == 0x6567612d && httpFold[key[8]] == 'n' && httpFold[key[9]] == 't' {
				req.UserAgent = val
			}
		case 0x6c742d78:
			if foldKeyU32(key, 4) == 0x61682d73 && httpFold[key[8]] == 's' && httpFold[key[9]] == 'h' {
				req.TLSHash = val
			}
		}
	case 12:
		if foldKeyU64(key, 0) == 0x69772d7063742d78 && foldKeyU32(key, 8) == 0x776f646e {
			if win, ok := parseTCPWindowHeader(val); ok {
				req.TCPWindow = win
				req.TCPWindowSet = 1
			}
		} else if foldKeyU64(key, 0) == 0x2d746e65746e6f63 && foldKeyU32(key, 8) == 0x65707974 {
			req.ContentType = val
		}
	case 14:
		if foldKeyU64(key, 0) == 0x2d746e65746e6f63 && foldKeyU32(key, 8) == 0x676e656c &&
			httpFold[key[12]] == 't' && httpFold[key[13]] == 'h' {
			cl, ok := parseContentLengthStrict(val)
			if !ok {
				return ErrInvalid
			}
			if *hFlags&http1flCLSet != 0 && *clValue != cl {
				return ErrInvalid
			}
			*hFlags |= http1flCLSet
			*clValue = cl
			req.ContentLength = cl
			req.HasContentLength = true
		}
	case 15:
		switch foldKeyU32(key, 0) {
		case 0x6f662d78:
			if foldKeyU64(key, 4) == 0x2d64656472617772 && httpFold[key[12]] == 'f' &&
				httpFold[key[13]] == 'o' && httpFold[key[14]] == 'r' {
				req.ClientIP = val
			}
		case 0x65636361:
			if key[6] == '-' && httpFold[key[7]] == 'l' {
				req.AcceptLang = val
			} else if key[6] == '-' && httpFold[key[7]] == 'e' {
				req.AcceptEncoding = val
			}
		}
	case 17:
		if foldKeyU64(key, 0) == 0x726566736e617274 && foldKeyU64(key, 8) == 0x6e69646f636e652d &&
			httpFold[key[16]] == 'g' {
			return http1AssignTransferEncoding(hFlags, val)
		}
	case 31:
		if http1MatchForceSafeHeader(key) && http1ForceSafeValue(val) {
			req.ForceSafe = true
		}
	}
	http1AssignWireMetadataHeaders(req, key, val)
	return nil
}
