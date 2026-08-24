package fraud


type ResidentialIntelResult struct {
	ResidentialProxy bool `json:"residential_proxy"`
	VPN              bool `json:"vpn"`
	Proxy            bool `json:"proxy"`
}


func (r ResidentialIntelResult) IsResidentialProxyFarm() bool {
	return r.ResidentialProxy || (r.Proxy && r.VPN)
}
