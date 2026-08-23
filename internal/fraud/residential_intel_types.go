package fraud

// ResidentialIntelResult is a normalized provider response for one IP lookup.
type ResidentialIntelResult struct {
	ResidentialProxy bool `json:"residential_proxy"`
	VPN              bool `json:"vpn"`
	Proxy            bool `json:"proxy"`
}

// IsResidentialProxyFarm returns true when external intel flags residential proxy traffic.
func (r ResidentialIntelResult) IsResidentialProxyFarm() bool {
	return r.ResidentialProxy || (r.Proxy && r.VPN)
}
