package nodeadmin

type MetricsHost interface {
	ScorerHost
	NodeIdentity() (nodeID, role string, region int16)
}
