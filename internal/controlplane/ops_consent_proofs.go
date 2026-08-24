package controlplane

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
)

func (r *opsReader) ListConsentProofs(ctx context.Context, userID, cursor string, limit int32) (ConsentProofListResult, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return ConsentProofListResult{}, fmt.Errorf("postgres pool not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return ConsentProofListResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}

	var hashFilter []byte
	userID = strings.TrimSpace(userID)
	if userID != "" {
		hashFilter = domain.HashUserID(userID)
	}

	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT ce.id, ce.user_id_hash, ce.purposes, ce.source, ce.created_at,
		 COALESCE(ucs.ad_storage, false), COALESCE(ucs.analytics_storage, false)
		FROM consent_events ce
		LEFT JOIN user_consent_state ucs ON ucs.user_id_hash = ce.user_id_hash
		WHERE ($1::bytea IS NULL OR ce.user_id_hash = $1)
		 AND ($2::bigint = 0 OR ce.id < $2)
		ORDER BY ce.id DESC
		LIMIT $3`, hashFilter, cursorID, limit+1)
	if err != nil {
		return ConsentProofListResult{}, err
	}
	defer rows.Close()

	var items []ConsentProofDTO
	for rows.Next() {
		var (
			id        int64
			hash      []byte
			purposes  int16
			source    string
			createdAt time.Time
			adStorage bool
			analytics bool
		)
		if err := rows.Scan(&id, &hash, &purposes, &source, &createdAt, &adStorage, &analytics); err != nil {
			return ConsentProofListResult{}, err
		}
		items = append(items, ConsentProofDTO{
			ID:         id,
			UserIDHash: hex.EncodeToString(hash),
			Purposes:   purposes,
			Source:     source,
			RecordedAt: createdAt.UTC().Format(time.RFC3339),
			AdStorage:  adStorage,
			Analytics:  analytics,
		})
	}
	if err := rows.Err(); err != nil {
		return ConsentProofListResult{}, err
	}

	result := ConsentProofListResult{Items: items}
	if int32(len(items)) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
	}
	if result.Items == nil {
		result.Items = []ConsentProofDTO{}
	}
	return result, nil
}
