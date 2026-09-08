package domain

type CrossLayerDesyncAction string

const (
	CrossLayerDesyncOff      CrossLayerDesyncAction = "off"
	CrossLayerDesyncBoost    CrossLayerDesyncAction = "boost"
	CrossLayerDesyncSafePage CrossLayerDesyncAction = "safe_page"
	CrossLayerDesyncBlock    CrossLayerDesyncAction = "block"
)

const (
	CrossLayerDesyncThresholdMin     uint8 = 2
	CrossLayerDesyncThresholdMax     uint8 = 5
	CrossLayerDesyncThresholdDefault uint8 = 3
)

func ParseCrossLayerDesyncAction(raw string) CrossLayerDesyncAction {
	switch raw {
	case string(CrossLayerDesyncOff):
		return CrossLayerDesyncOff
	case string(CrossLayerDesyncSafePage):
		return CrossLayerDesyncSafePage
	case string(CrossLayerDesyncBlock):
		return CrossLayerDesyncBlock
	default:
		return CrossLayerDesyncBoost
	}
}

func (a CrossLayerDesyncAction) Valid() bool {
	switch a {
	case CrossLayerDesyncOff, CrossLayerDesyncBoost, CrossLayerDesyncSafePage, CrossLayerDesyncBlock:
		return true
	default:
		return false
	}
}

func NormalizeCrossLayerDesyncThreshold(threshold uint8) uint8 {
	if threshold < CrossLayerDesyncThresholdMin {
		return CrossLayerDesyncThresholdDefault
	}
	if threshold > CrossLayerDesyncThresholdMax {
		return CrossLayerDesyncThresholdMax
	}
	return threshold
}
