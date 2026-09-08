package flow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const landerSelectWithVersions = `
SELECT
	l.id,
	l.name,
	COALESCE(l.url, ''),
	l.hosted_asset_id,
	l.created_at,
	COALESCE((SELECT MAX(la.version) FROM lander_assets la WHERE la.lander_id = l.id), 0) AS draft_version,
	COALESCE(pub.version, 0) AS published_version
FROM landers l
LEFT JOIN lander_assets pub ON pub.id = l.hosted_asset_id`

type UpdateLanderRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func applyLanderVersionFields(dto *LanderDTO, draftVersion, publishedVersion int) {
	if dto == nil {
		return
	}
	dto.DraftVersion = draftVersion
	dto.PublishedVersion = publishedVersion
	dto.HasUnpublishedDraft = draftVersion > publishedVersion && draftVersion > 0
}

func finishLanderDTO(dto *LanderDTO, publicBase string, draftVersion, publishedVersion int) {
	applyLanderVersionFields(dto, draftVersion, publishedVersion)
	if dto.HostedAssetID != nil {
		dto.HostedURL = landerhost.PublicURL(publicBase, dto.ID)
	}
}

func (st *Store) scanLanderRow(row pgx.Row, publicBase string) (LanderDTO, error) {
	var dto LanderDTO
	var draftVersion int
	var publishedVersion int
	err := row.Scan(
		&dto.ID,
		&dto.Name,
		&dto.URL,
		&dto.HostedAssetID,
		&dto.CreatedAt,
		&draftVersion,
		&publishedVersion,
	)
	if err != nil {
		return LanderDTO{}, err
	}
	finishLanderDTO(&dto, publicBase, draftVersion, publishedVersion)
	return dto, nil
}

func (st *Store) GetLander(ctx context.Context, landerID uuid.UUID) (LanderDTO, error) {
	if st.poolOrNil() == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	if landerID == uuid.Nil {
		return LanderDTO{}, fmt.Errorf("lander id is required")
	}
	publicBase := st.host.LanderPublicBase(ctx)
	row := st.poolOrNil().QueryRow(ctx, landerSelectWithVersions+` WHERE l.id = $1`, landerID)
	dto, err := st.scanLanderRow(row, publicBase)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LanderDTO{}, fmt.Errorf("lander not found")
		}
		return LanderDTO{}, err
	}
	return dto, nil
}

func (st *Store) UpdateLander(ctx context.Context, landerID uuid.UUID, req UpdateLanderRequest) (LanderDTO, error) {
	if st.poolOrNil() == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	if landerID == uuid.Nil {
		return LanderDTO{}, fmt.Errorf("lander id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return LanderDTO{}, fmt.Errorf("name is required")
	}
	url := strings.TrimSpace(req.URL)

	current, err := st.GetLander(ctx, landerID)
	if err != nil {
		return LanderDTO{}, err
	}

	if current.HostedAssetID != nil {
		tag, err := st.poolOrNil().Exec(ctx, `UPDATE landers SET name = $2 WHERE id = $1`, landerID, name)
		if err != nil {
			return LanderDTO{}, err
		}
		if tag.RowsAffected() == 0 {
			return LanderDTO{}, fmt.Errorf("lander not found")
		}
		return st.GetLander(ctx, landerID)
	}

	tag, err := st.poolOrNil().Exec(ctx, `
		UPDATE landers SET name = $2, url = NULLIF($3, '')
		WHERE id = $1`, landerID, name, url)
	if err != nil {
		return LanderDTO{}, err
	}
	if tag.RowsAffected() == 0 {
		return LanderDTO{}, fmt.Errorf("lander not found")
	}
	return st.GetLander(ctx, landerID)
}

func (st *Store) DeleteLander(ctx context.Context, landerID uuid.UUID) error {
	if st.poolOrNil() == nil {
		return fmt.Errorf("service unavailable")
	}
	if landerID == uuid.Nil {
		return fmt.Errorf("lander id is required")
	}
	referenced, err := st.landerReferencedByFlow(ctx, landerID)
	if err != nil {
		return err
	}
	if referenced {
		return fmt.Errorf("lander is referenced by a flow")
	}
	tag, err := st.poolOrNil().Exec(ctx, `DELETE FROM landers WHERE id = $1`, landerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lander not found")
	}
	return nil
}

func (st *Store) landerReferencedByFlow(ctx context.Context, landerID uuid.UUID) (bool, error) {
	var referenced bool
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM flows f,
				jsonb_array_elements(f.paths) AS path,
				jsonb_array_elements(path->'landers') AS lr
			WHERE lr->>'lander_id' = $1::text
		)`, landerID).Scan(&referenced)
	return referenced, err
}
