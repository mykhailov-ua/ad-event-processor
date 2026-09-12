package flow

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/pkg/coldpath"
)

const (
	landerListDefaultLimit = 25
	landerListMaxLimit     = 1000
)

type ListLandersFilter struct {
	Search  string
	Hosting string
	Limit   int32
	Offset  int32
}

type LanderHostingCountsDTO struct {
	Total        int64 `json:"total"`
	External     int64 `json:"external"`
	Hosted       int64 `json:"hosted"`
	Unconfigured int64 `json:"unconfigured"`
}

type LanderListResponse struct {
	Items         []LanderDTO            `json:"items"`
	Total         int64                  `json:"total"`
	Limit         int32                  `json:"limit"`
	Offset        int32                  `json:"offset"`
	HostingCounts LanderHostingCountsDTO `json:"hosting_counts"`
}

func buildLanderListWhere(filter ListLandersFilter) (string, []any) {
	var clauses []string
	var args []any
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	switch strings.TrimSpace(filter.Hosting) {
	case "hosted":
		clauses = append(clauses, "l.hosted_asset_id IS NOT NULL")
	case "external":
		clauses = append(clauses, "l.hosted_asset_id IS NULL AND COALESCE(l.url, '') <> ''")
	}

	search := strings.TrimSpace(filter.Search)
	if search != "" {
		pattern := "%" + search + "%"
		nameArg := nextArg(pattern)
		idArg := nextArg(pattern)
		urlArg := nextArg(pattern)
		clauses = append(clauses, fmt.Sprintf(
			"(l.name ILIKE %s OR l.id::text ILIKE %s OR COALESCE(l.url, '') ILIKE %s)",
			nameArg,
			idArg,
			urlArg,
		))
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (st *Store) countLanderHostingTotals(ctx context.Context) (LanderHostingCountsDTO, error) {
	var counts LanderHostingCountsDTO
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT
			COUNT(*)::bigint,
			COUNT(*) FILTER (WHERE l.hosted_asset_id IS NULL AND COALESCE(l.url, '') <> '')::bigint,
			COUNT(*) FILTER (WHERE l.hosted_asset_id IS NOT NULL)::bigint,
			COUNT(*) FILTER (WHERE l.hosted_asset_id IS NULL AND COALESCE(l.url, '') = '')::bigint
		FROM landers l`).Scan(&counts.Total, &counts.External, &counts.Hosted, &counts.Unconfigured)
	return counts, err
}

func (st *Store) ListLandersPage(ctx context.Context, filter ListLandersFilter) (LanderListResponse, error) {
	if st.poolOrNil() == nil {
		return LanderListResponse{}, fmt.Errorf("service unavailable")
	}

	limit, offset := coldpath.ClampLimitOffset(filter.Limit, filter.Offset, landerListDefaultLimit, landerListMaxLimit)

	hostingCounts, err := st.countLanderHostingTotals(ctx)
	if err != nil {
		return LanderListResponse{}, err
	}

	where, args := buildLanderListWhere(filter)

	var total int64
	countSQL := `SELECT COUNT(*)::bigint FROM landers l` + where
	if err := st.poolOrNil().QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return LanderListResponse{}, err
	}

	if total == 0 {
		return LanderListResponse{
			Items:         []LanderDTO{},
			Total:         0,
			Limit:         limit,
			Offset:        offset,
			HostingCounts: hostingCounts,
		}, nil
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, limit, offset)
	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	listSQL := landerSelectWithVersions + where + fmt.Sprintf(
		` ORDER BY l.created_at DESC LIMIT $%d OFFSET $%d`,
		limitIdx,
		offsetIdx,
	)

	rows, err := st.poolOrNil().Query(ctx, listSQL, listArgs...)
	if err != nil {
		return LanderListResponse{}, err
	}
	defer rows.Close()

	publicBase := st.host.LanderPublicBase(ctx)
	items := make([]LanderDTO, 0, limit)
	for rows.Next() {
		dto, err := st.scanLanderRow(rows, publicBase)
		if err != nil {
			return LanderListResponse{}, err
		}
		items = append(items, dto)
	}
	if err := rows.Err(); err != nil {
		return LanderListResponse{}, err
	}

	return LanderListResponse{
		Items:         items,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
		HostingCounts: hostingCounts,
	}, nil
}

func (st *Store) ListLanders(ctx context.Context) ([]LanderDTO, error) {
	page, err := st.ListLandersPage(ctx, ListLandersFilter{
		Limit:  landerListMaxLimit,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}
