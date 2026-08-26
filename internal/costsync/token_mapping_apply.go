package costsync

import (
	"strings"
)

// ApplyNetworkObjectToken sets PlacementID from the configured network object for attribution.
func ApplyNetworkObjectToken(lines []CostLine, mapping TokenMapping) {
	obj := strings.ToLower(strings.TrimSpace(mapping.NetworkObject))
	if obj == "" || obj == "placement_id" {
		return
	}
	for i := range lines {
		switch obj {
		case "adset_id":
			if lines[i].AdsetID != "" {
				lines[i].PlacementID = lines[i].AdsetID
			}
		case "ad_id":
			if lines[i].AdID != "" {
				lines[i].PlacementID = lines[i].AdID
			}
		}
	}
}
