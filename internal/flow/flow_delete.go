package flow

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (st *Store) DeleteFlow(ctx context.Context, flowID uuid.UUID) error {
	if st.poolOrNil() == nil {
		return fmt.Errorf("service unavailable")
	}
	if flowID == uuid.Nil {
		return fmt.Errorf("flow id is required")
	}
	referenced, err := st.flowReferencedByCampaign(ctx, flowID)
	if err != nil {
		return err
	}
	if referenced {
		return fmt.Errorf("flow is referenced by a campaign")
	}
	tag, err := st.poolOrNil().Exec(ctx, `DELETE FROM flows WHERE id = $1`, flowID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("flow not found")
	}
	return nil
}

func (st *Store) flowReferencedByCampaign(ctx context.Context, flowID uuid.UUID) (bool, error) {
	var referenced bool
	err := st.poolOrNil().QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM campaigns
			WHERE flow_id = $1 AND deleted_at IS NULL
		)`, flowID).Scan(&referenced)
	return referenced, err
}
