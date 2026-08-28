package fraudadmin

import "github.com/jackc/pgx/v5/pgxpool"

type LabelsHost interface {
	LabelsPool() *pgxpool.Pool
}

type Labels struct {
	host LabelsHost
}

func NewLabels(host LabelsHost) *Labels {
	return &Labels{host: host}
}
