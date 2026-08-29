package parser

func AppendJSONString(dst []byte, s []byte) []byte {
	dst = append(dst, '"')
	for _, b := range s {
		switch b {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, b)
		}
	}
	dst = append(dst, '"')
	return dst
}

func MarshalExtra(dst []byte, keys, values [][]byte) []byte {
	dst = dst[:0]
	dst = append(dst, '{')
	for i, key := range keys {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = AppendJSONString(dst, key)
		dst = append(dst, ':')
		if i < len(values) {
			dst = AppendJSONString(dst, values[i])
		} else {
			dst = append(dst, '"', '"')
		}
	}
	dst = append(dst, '}')
	return dst
}
