package track

func ParseDmrQueryFlag(decoded []byte) bool {
	n := len(decoded)
	if n == 1 && decoded[0] == '1' {
		return true
	}
	if n != 4 {
		return false
	}
	return (decoded[0] == 't' || decoded[0] == 'T') &&
		(decoded[1] == 'r' || decoded[1] == 'R') &&
		(decoded[2] == 'u' || decoded[2] == 'U') &&
		(decoded[3] == 'e' || decoded[3] == 'E')
}
