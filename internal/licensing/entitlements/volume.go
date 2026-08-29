package entitlements

type VolumeBand string

const (
	VolumeBandSmall  VolumeBand = "S"
	VolumeBandMedium VolumeBand = "M"
	VolumeBandLarge  VolumeBand = "L"
)

type BillableCategory uint8

const (
	BillableAccepted BillableCategory = iota
	BillableDedupReject
	BillableEbpfDrop
)

const (
	weightAccepted    = 1.0
	weightDedupReject = 0.1
	weightEbpfDrop    = 0.0
)

var BandIncludedEvents = map[VolumeBand]uint64{
	VolumeBandSmall:  10_000_000_000,
	VolumeBandMedium: 50_000_000_000,
	VolumeBandLarge:  100_000_000_000,
}

var BasePU = map[VolumeBand]int{
	VolumeBandSmall:  100,
	VolumeBandMedium: 250,
	VolumeBandLarge:  500,
}

type ModulePU struct {
	OpenRTBEngine int
	EbpfXDPEdge   int
	IvtMLDetector int
	MlFraudBoost  int
}

var ModuleCoefficients = map[VolumeBand]ModulePU{
	VolumeBandSmall:  {OpenRTBEngine: 50, EbpfXDPEdge: 40, IvtMLDetector: 40, MlFraudBoost: 30},
	VolumeBandMedium: {OpenRTBEngine: 120, EbpfXDPEdge: 100, IvtMLDetector: 80, MlFraudBoost: 60},
	VolumeBandLarge:  {OpenRTBEngine: 250, EbpfXDPEdge: 200, IvtMLDetector: 150, MlFraudBoost: 100},
}

func BillableWeight(cat BillableCategory) float64 {
	switch cat {
	case BillableAccepted:
		return weightAccepted
	case BillableDedupReject:
		return weightDedupReject
	case BillableEbpfDrop:
		return weightEbpfDrop
	default:
		return weightAccepted
	}
}

func BillableWeightPermille(cat BillableCategory) int64 {
	switch cat {
	case BillableAccepted:
		return 1000
	case BillableDedupReject:
		return 100
	case BillableEbpfDrop:
		return 0
	default:
		return 1000
	}
}

func ClassifyEventType(eventType string) BillableCategory {
	switch eventType {
	case "duplicate", "dedup", "dedup_reject", "freq", "fcap", "rate_limit":
		return BillableDedupReject
	case "ebpf_drop", "l3_blocklist", "tls_blocklist", "xdp_drop":
		return BillableEbpfDrop
	default:
		return BillableAccepted
	}
}

func WeightedBillableUnits(counts map[BillableCategory]uint64) float64 {
	var total float64
	for cat, n := range counts {
		total += float64(n) * BillableWeight(cat)
	}
	return total
}

func MonthlyPU(band VolumeBand, features FeatureSet) int {
	if band == "" {
		band = VolumeBandSmall
	}
	pu := BasePU[band]
	mods := ModuleCoefficients[band]
	features = features.Normalized()
	if features.OpenRTBEnabled() {
		pu += mods.OpenRTBEngine
	}
	if features.EbpfXDPEdge {
		pu += mods.EbpfXDPEdge
	}
	if features.IvtMLDetector {
		pu += mods.IvtMLDetector
	}
	if features.MlFraudBoostEnabled() {
		pu += mods.MlFraudBoost
	}
	return pu
}

func ParseVolumeBand(raw string) VolumeBand {
	switch VolumeBand(raw) {
	case VolumeBandSmall, VolumeBandMedium, VolumeBandLarge:
		return VolumeBand(raw)
	default:
		return VolumeBandSmall
	}
}
