package fraudadmin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/moderatorcorpus"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	ModeratorCorpusDefaultLimit = 50
	ModeratorCorpusMaxLimit     = 200
)

type ModeratorCorpusDTO struct {
	ID               string `json:"id"`
	JA3              string `json:"ja3"`
	JA4              string `json:"ja4,omitempty"`
	TCPSig           string `json:"tcp_sig,omitempty"`
	WebGLRenderer    string `json:"webgl_renderer,omitempty"`
	LayerDesyncCount int    `json:"layer_desync_count"`
	Note             string `json:"note,omitempty"`
	Source           string `json:"source"`
	CreatedAt        string `json:"created_at"`
	CreatedAtDisplay string `json:"created_at_display,omitempty"`
	UpdatedAt        string `json:"updated_at"`
	UpdatedAtDisplay string `json:"updated_at_display,omitempty"`
}

type ModeratorCorpusListResponse struct {
	Items       []ModeratorCorpusDTO `json:"items"`
	Total       int64                `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
	LastRefresh string               `json:"last_refresh,omitempty"`
}

type ModeratorCorpusUpsertRequest struct {
	JA3              string `json:"ja3"`
	JA4              string `json:"ja4,omitempty"`
	TCPSig           string `json:"tcp_sig,omitempty"`
	WebGLRenderer    string `json:"webgl_renderer,omitempty"`
	LayerDesyncCount int    `json:"layer_desync_count,omitempty"`
	Note             string `json:"note,omitempty"`
	Source           string `json:"source,omitempty"`
}

type ModeratorCorpusImportRequest struct {
	CSV string `json:"csv"`
}

type ModeratorCorpusImportResponse struct {
	Upserted int `json:"upserted"`
}

type ModeratorCorpusPreviewResponse struct {
	MatchCount7d int64 `json:"match_count_7d"`
}

func normalizeModeratorCorpusLimit(limit int) int {
	if limit <= 0 {
		return ModeratorCorpusDefaultLimit
	}
	if limit > ModeratorCorpusMaxLimit {
		return ModeratorCorpusMaxLimit
	}
	return limit
}

func (m *ModeratorCorpus) ListTuples(ctx context.Context, limit, offset int) ([]ModeratorCorpusDTO, int64, error) {
	if m == nil || m.host == nil || m.host.ModeratorCorpusPool() == nil {
		return nil, 0, fmt.Errorf("postgres pool not configured")
	}
	limit = normalizeModeratorCorpusLimit(limit)
	if offset < 0 {
		offset = 0
	}
	pool := m.host.ModeratorCorpusPool()
	var total int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fraud_moderator_corpus`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count fraud_moderator_corpus: %w", err)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count, note, source, created_at, updated_at
		FROM fraud_moderator_corpus
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query fraud_moderator_corpus: %w", err)
	}
	defer rows.Close()
	out := make([]ModeratorCorpusDTO, 0, limit)
	for rows.Next() {
		dto, err := scanModeratorCorpusRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, dto)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (m *ModeratorCorpus) UpsertTuple(ctx context.Context, req ModeratorCorpusUpsertRequest) (ModeratorCorpusDTO, error) {
	tuple, err := moderatorcorpus.NormalizeTuple(moderatorcorpus.Tuple{
		JA3:              req.JA3,
		JA4:              req.JA4,
		TCPSig:           req.TCPSig,
		WebGLRenderer:    req.WebGLRenderer,
		LayerDesyncCount: uint8(req.LayerDesyncCount),
	})
	if err != nil {
		return ModeratorCorpusDTO{}, ValidationError(err.Error())
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "manual"
	}
	note := strings.TrimSpace(req.Note)
	if m == nil || m.host == nil || m.host.ModeratorCorpusPool() == nil {
		return ModeratorCorpusDTO{}, fmt.Errorf("postgres pool not configured")
	}
	var id uuid.UUID
	var createdAt, updatedAt time.Time
	err = m.host.ModeratorCorpusPool().QueryRow(ctx, `
		INSERT INTO fraud_moderator_corpus (ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count, note, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
		ON CONFLICT (ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count)
		DO UPDATE SET note = EXCLUDED.note, source = EXCLUDED.source, updated_at = now()
		RETURNING id, created_at, updated_at`,
		tuple.JA3, tuple.JA4, tuple.TCPSig, tuple.WebGLRenderer, tuple.LayerDesyncCount, note, source,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return ModeratorCorpusDTO{}, fmt.Errorf("upsert fraud_moderator_corpus: %w", err)
	}
	if err := m.host.RefreshModeratorCorpusFeed(ctx); err != nil {
		return ModeratorCorpusDTO{}, err
	}
	return ModeratorCorpusDTO{
		ID:               id.String(),
		JA3:              tuple.JA3,
		JA4:              tuple.JA4,
		TCPSig:           tuple.TCPSig,
		WebGLRenderer:    tuple.WebGLRenderer,
		LayerDesyncCount: int(tuple.LayerDesyncCount),
		Note:             note,
		Source:           source,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		CreatedAtDisplay: coldpath.RFC3339Display(createdAt.UTC().Format(time.RFC3339)),
		UpdatedAt:        updatedAt.UTC().Format(time.RFC3339),
		UpdatedAtDisplay: coldpath.RFC3339Display(updatedAt.UTC().Format(time.RFC3339)),
	}, nil
}

func (m *ModeratorCorpus) ImportCSV(ctx context.Context, csvBody string) (int, error) {
	rows, err := parseModeratorCorpusCSV(csvBody)
	if err != nil {
		return 0, ValidationError(err.Error())
	}
	if len(rows) == 0 {
		return 0, ValidationError("csv has no data rows")
	}
	if m == nil || m.host == nil || m.host.ModeratorCorpusPool() == nil {
		return 0, fmt.Errorf("postgres pool not configured")
	}
	upserted := 0
	for _, row := range rows {
		tuple, err := moderatorcorpus.NormalizeTuple(moderatorcorpus.Tuple{
			JA3:              row.JA3,
			JA4:              row.JA4,
			TCPSig:           row.TCPSig,
			WebGLRenderer:    row.WebGLRenderer,
			LayerDesyncCount: uint8(row.LayerDesyncCount),
		})
		if err != nil {
			return upserted, ValidationError(err.Error())
		}
		source := strings.TrimSpace(row.Source)
		if source == "" {
			source = "import"
		}
		note := strings.TrimSpace(row.Note)
		if _, err := m.host.ModeratorCorpusPool().Exec(ctx, `
			INSERT INTO fraud_moderator_corpus (ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count, note, source, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now())
			ON CONFLICT (ja3, ja4, tcp_sig, webgl_renderer, layer_desync_count)
			DO UPDATE SET note = EXCLUDED.note, source = EXCLUDED.source, updated_at = now()`,
			tuple.JA3, tuple.JA4, tuple.TCPSig, tuple.WebGLRenderer, tuple.LayerDesyncCount, note, source,
		); err != nil {
			return upserted, fmt.Errorf("import fraud_moderator_corpus: %w", err)
		}
		upserted++
	}
	if err := m.host.RefreshModeratorCorpusFeed(ctx); err != nil {
		return upserted, err
	}
	return upserted, nil
}

func (m *ModeratorCorpus) PreviewMatchCount7d(ctx context.Context, ja3 string) (int64, error) {
	ja3 = strings.TrimSpace(ja3)
	if ja3 == "" {
		return 0, ValidationError("ja3 is required")
	}
	ch := m.host.ModeratorCorpusClickHouse()
	if ch == nil {
		return 0, nil
	}
	var count int64
	err := ch.QueryRow(ctx, `
		SELECT count()
		FROM clicks
		WHERE review_routed_event = 1
		  AND created_at >= now() - INTERVAL 7 DAY
		  AND tls_hash = ?`, ja3).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("preview moderator corpus matches: %w", err)
	}
	return count, nil
}

func parseModeratorCorpusCSV(body string) ([]ModeratorCorpusUpsertRequest, error) {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty csv")
	}
	start := 0
	header := strings.ToLower(strings.TrimSpace(lines[0]))
	if strings.Contains(header, "ja3") {
		start = 1
	}
	out := make([]ModeratorCorpusUpsertRequest, 0, len(lines)-start)
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := splitCSVLine(line)
		if len(parts) == 0 {
			continue
		}
		req := ModeratorCorpusUpsertRequest{JA3: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			req.JA4 = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			req.TCPSig = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			req.WebGLRenderer = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 {
			n, err := parseCSVInt(parts[4])
			if err != nil {
				return nil, fmt.Errorf("row %d: invalid layer_desync_count", i+1)
			}
			req.LayerDesyncCount = n
		}
		if len(parts) > 5 {
			req.Note = strings.TrimSpace(parts[5])
		}
		if _, err := moderatorcorpus.NormalizeTuple(moderatorcorpus.Tuple{
			JA3:              req.JA3,
			JA4:              req.JA4,
			TCPSig:           req.TCPSig,
			WebGLRenderer:    req.WebGLRenderer,
			LayerDesyncCount: uint8(req.LayerDesyncCount),
		}); err != nil {
			return nil, fmt.Errorf("row %d: %s", i+1, err.Error())
		}
		out = append(out, req)
	}
	return out, nil
}

func splitCSVLine(line string) []string {
	parts := strings.Split(line, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if len(parts[i]) >= 2 && parts[i][0] == '"' && parts[i][len(parts[i])-1] == '"' {
			parts[i] = parts[i][1 : len(parts[i])-1]
		}
	}
	return parts
}

func parseCSVInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	var n int
	_, err := fmt.Sscanf(raw, "%d", &n)
	if err != nil || n < 0 || n > 255 {
		return 0, fmt.Errorf("invalid int")
	}
	return n, nil
}

func scanModeratorCorpusRow(rows pgx.Rows) (ModeratorCorpusDTO, error) {
	var dto ModeratorCorpusDTO
	var id uuid.UUID
	var desync int16
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&id, &dto.JA3, &dto.JA4, &dto.TCPSig, &dto.WebGLRenderer, &desync, &dto.Note, &dto.Source, &createdAt, &updatedAt); err != nil {
		return ModeratorCorpusDTO{}, err
	}
	dto.ID = id.String()
	dto.LayerDesyncCount = int(desync)
	dto.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	dto.CreatedAtDisplay = coldpath.RFC3339Display(dto.CreatedAt)
	dto.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	dto.UpdatedAtDisplay = coldpath.RFC3339Display(dto.UpdatedAt)
	return dto, nil
}
