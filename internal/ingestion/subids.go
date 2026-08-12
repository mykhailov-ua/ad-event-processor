package ingestion

const MaxSubIDs = 30

type SubIDSlots [MaxSubIDs]string

func (s *SubIDSlots) reset() {
	for i := range s {
		s[i] = ""
	}
}

// subKeyIndex returns 1-based sub index for keys sub1..sub30.
func subKeyIndex(key []byte) (int, bool) {
	n := len(key)
	if n < 4 || n > 5 {
		return 0, false
	}
	if key[0] != 's' || key[1] != 'u' || key[2] != 'b' {
		return 0, false
	}
	if n == 4 {
		if key[3] < '1' || key[3] > '9' {
			return 0, false
		}
		return int(key[3] - '0'), true
	}
	if key[3] < '1' || key[3] > '3' {
		return 0, false
	}
	if key[4] < '0' || key[4] > '9' {
		return 0, false
	}
	idx := int(key[3]-'0')*10 + int(key[4]-'0')
	if idx < 10 || idx > MaxSubIDs {
		return 0, false
	}
	return idx, true
}
