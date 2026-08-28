package dashboardadmin

type MLManualLabelDTO struct {
	IPHash    string `json:"ip_hash"`
	Label     int    `json:"label"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type FraudTierThresholdsDTO struct {
	Scope      string `json:"scope,omitempty"`
	PassMax    int    `json:"pass_max"`
	SuspectMax int    `json:"suspect_max"`
	IVTMax     int    `json:"ivt_max"`
	BlockAbove int    `json:"block_above"`
}
