package rtb

import (
	"errors"
	"strings"
)

var ErrInvalidDealPacing = errors.New("pacing must be open or closed")

func ParseDealPacingString(v string) (int16, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "open":
		return int16(PacingOpen), nil
	case "closed":
		return int16(PacingClosed), nil
	default:
		return 0, ErrInvalidDealPacing
	}
}

func DealPacingLabel(p int16) string {
	if p == int16(PacingClosed) {
		return "closed"
	}
	return "open"
}

func DealPacingOpen(p int16) uint8 {
	if p == int16(PacingClosed) {
		return PacingClosed
	}
	return PacingOpen
}
