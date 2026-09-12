package entitlements

// Cost sync credential limits use MaxCostSyncNetworks on deployment Limits:
//   0 = API credential connect and automated run disabled (manual cost entry only)
//   1..999998 = max distinct networks per customer
//   >= 999999 = unlimited API networks

func CostSyncNetworksDisabled(limits Limits) bool {
	return limits.MaxCostSyncNetworks == 0
}

func CostSyncNetworksUnlimited(limits Limits) bool {
	return limits.MaxCostSyncNetworks >= 999999
}

func CostSyncNetworksCap(limits Limits) uint64 {
	if CostSyncNetworksDisabled(limits) || CostSyncNetworksUnlimited(limits) {
		return 0
	}
	return limits.MaxCostSyncNetworks
}
