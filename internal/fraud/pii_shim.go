package fraud

import "ad-event-processor/internal/fraud/features"

const emptyIPHashFilter = features.EmptyIPHashFilter

func hashIPForClickhouse(ip string) [16]byte {
	return features.HashIPForClickhouse(ip)
}
