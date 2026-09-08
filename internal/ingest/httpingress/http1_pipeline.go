package httpingress

import "errors"

// CountCompleteHTTP1Messages returns fully parsed HTTP/1 messages in data.
// Stops at the first ErrIncomplete or any non-incomplete parse error.
func CountCompleteHTTP1Messages(data []byte, maxBody int64, scratchPtr *[]byte, limits ParseLimits) int {
	count := 0
	offset := 0
	var req Request
	for offset < len(data) {
		n, err := ParseHTTP1LimitsInto(data[offset:], maxBody, scratchPtr, limits, &req)
		if err != nil {
			if errors.Is(err, ErrIncomplete) {
				break
			}
			break
		}
		if n <= 0 {
			break
		}
		count++
		offset += n
	}
	return count
}
