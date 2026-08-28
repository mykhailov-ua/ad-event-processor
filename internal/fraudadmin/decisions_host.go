package fraudadmin

import (
	"context"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/fraud"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DecisionsHost interface {
	DecisionsPool() *pgxpool.Pool
	DecisionsClickHouse() *database.ClickHouseQuery
	FraudExplainLiveScoreEnabled() bool
	FraudExplainScorer(ctx context.Context) (fraud.Scorer, error)
}
