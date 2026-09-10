package teamscope

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamDTO struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at,omitempty"`
}

func ListTeams(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]TeamDTO, error) {
	if pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	q := db.New(pool)
	rows, err := q.ListTeamsByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	out := make([]TeamDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, teamRowToDTO(row))
	}
	return out, nil
}

func CreateTeam(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, name string) (TeamDTO, error) {
	if pool == nil {
		return TeamDTO{}, fmt.Errorf("service unavailable")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return TeamDTO{}, fmt.Errorf("name is required")
	}
	q := db.New(pool)
	row, err := q.CreateTeam(ctx, db.CreateTeamParams{
		ID:         domain.ToUUID(uuid.New()),
		CustomerID: domain.ToUUID(customerID),
		Name:       name,
	})
	if err != nil {
		return TeamDTO{}, err
	}
	return teamRowToDTO(row), nil
}

func teamRowToDTO(row db.Team) TeamDTO {
	id := uuid.UUID(row.ID.Bytes)
	customerID := uuid.UUID(row.CustomerID.Bytes)
	createdAt := ""
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return TeamDTO{
		ID:         id.String(),
		CustomerID: customerID.String(),
		Name:       row.Name,
		CreatedAt:  createdAt,
	}
}
