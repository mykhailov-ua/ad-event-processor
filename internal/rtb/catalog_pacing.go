package rtb

const (
	PacingOpen   uint8 = 1
	PacingClosed uint8 = 2
)

func normalizePacingOpen(open uint8) uint8 {
	if open == PacingClosed {
		return PacingClosed
	}
	return PacingOpen
}
