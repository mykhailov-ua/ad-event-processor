package reports

type EdgeMetricsPanelDTO struct {
	UpdatedAt      string            `json:"updated_at,omitempty"`
	IngressH1      uint64            `json:"ingress_h1"`
	IngressH2      uint64            `json:"ingress_h2"`
	IngressH3      uint64            `json:"ingress_h3"`
	BodyStream     uint64            `json:"body_stream"`
	BodyPeek       uint64            `json:"body_peek"`
	BodyRead       uint64            `json:"body_read"`
	Blocked        map[string]uint64 `json:"blocked"`
	TarpitTotal    uint64            `json:"tarpit_total"`
	BlacklistStale uint64            `json:"blacklist_stale"`
}
