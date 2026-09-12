package campaign

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCampaignGroupNotFound = errors.New("campaign group not found")

type CampaignGroupDTO struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	Name          string `json:"name"`
	DefaultFlowID string `json:"default_flow_id,omitempty"`
	MemberCount   int64  `json:"member_count,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CreateCampaignGroupRequest struct {
	CustomerID    string `json:"customer_id"`
	Name          string `json:"name"`
	DefaultFlowID string `json:"default_flow_id,omitempty"`
}

type UpdateCampaignGroupRequest struct {
	Name          *string `json:"name,omitempty"`
	DefaultFlowID *string `json:"default_flow_id,omitempty"`
}

type AssignCampaignGroupMembersRequest struct {
	CampaignIDs []string `json:"campaign_ids"`
}

type CampaignGroupAssignResultRow struct {
	CampaignID string `json:"campaign_id"`
	OK         bool   `json:"ok"`
	ErrorCode  string `json:"error_code,omitempty"`
}

type AssignCampaignGroupMembersResponse struct {
	Results []CampaignGroupAssignResultRow `json:"results"`
}

func campaignGroupToDTO(row db.CampaignGroup, memberCount int64) CampaignGroupDTO {
	dto := CampaignGroupDTO{
		ID:          uuid.UUID(row.ID.Bytes).String(),
		CustomerID:  uuid.UUID(row.CustomerID.Bytes).String(),
		Name:        row.Name,
		MemberCount: memberCount,
		CreatedAt:   row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
	if row.DefaultFlowID.Valid {
		dto.DefaultFlowID = uuid.UUID(row.DefaultFlowID.Bytes).String()
	}
	return dto
}

func mapCampaignGroupStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCampaignGroupNotFound
	}
	if IsPgUniqueViolation(err) {
		return errValidation("campaign group name already exists for customer")
	}
	return err
}

func ListCampaignGroups(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]CampaignGroupDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	rows, err := db.New(pool).ListCampaignGroupsByCustomer(ctx, domain.ToUUID(customerID))
	if err != nil {
		return nil, err
	}
	q := db.New(pool)
	out := make([]CampaignGroupDTO, 0, len(rows))
	for i := range rows {
		groupID := uuid.UUID(rows[i].ID.Bytes)
		count, err := q.CountCampaignsByGroup(ctx, db.CountCampaignsByGroupParams{
			CustomerID:      domain.ToUUID(customerID),
			CampaignGroupID: domain.ToUUID(groupID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, campaignGroupToDTO(rows[i], count))
	}
	return out, nil
}

func GetCampaignGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (CampaignGroupDTO, error) {
	if pool == nil {
		return CampaignGroupDTO{}, errServiceUnavailable()
	}
	q := db.New(pool)
	row, err := q.GetCampaignGroup(ctx, domain.ToUUID(groupID))
	if err != nil {
		return CampaignGroupDTO{}, mapCampaignGroupStoreError(err)
	}
	count, err := q.CountCampaignsByGroup(ctx, db.CountCampaignsByGroupParams{
		CustomerID:      row.CustomerID,
		CampaignGroupID: row.ID,
	})
	if err != nil {
		return CampaignGroupDTO{}, err
	}
	return campaignGroupToDTO(row, count), nil
}

func CreateCampaignGroup(ctx context.Context, pool *pgxpool.Pool, req CreateCampaignGroupRequest) (CampaignGroupDTO, error) {
	if pool == nil {
		return CampaignGroupDTO{}, errServiceUnavailable()
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CampaignGroupDTO{}, errValidation("name is required")
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		return CampaignGroupDTO{}, errValidation("invalid customer_id")
	}
	var defaultFlow pgtype.UUID
	if flowRaw := strings.TrimSpace(req.DefaultFlowID); flowRaw != "" {
		flowID, err := uuid.Parse(flowRaw)
		if err != nil {
			return CampaignGroupDTO{}, errValidation("invalid default_flow_id")
		}
		defaultFlow = domain.ToUUID(flowID)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return CampaignGroupDTO{}, err
	}
	row, err := db.New(pool).InsertCampaignGroup(ctx, db.InsertCampaignGroupParams{
		ID:            domain.ToUUID(id),
		CustomerID:    domain.ToUUID(customerID),
		Name:          name,
		DefaultFlowID: defaultFlow,
	})
	if err != nil {
		return CampaignGroupDTO{}, mapCampaignGroupStoreError(err)
	}
	return campaignGroupToDTO(row, 0), nil
}

func UpdateCampaignGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, req UpdateCampaignGroupRequest) (CampaignGroupDTO, error) {
	if pool == nil {
		return CampaignGroupDTO{}, errServiceUnavailable()
	}
	current, err := db.New(pool).GetCampaignGroup(ctx, domain.ToUUID(groupID))
	if err != nil {
		return CampaignGroupDTO{}, mapCampaignGroupStoreError(err)
	}
	name := current.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return CampaignGroupDTO{}, errValidation("name is required")
		}
	}
	defaultFlow := current.DefaultFlowID
	if req.DefaultFlowID != nil {
		flowRaw := strings.TrimSpace(*req.DefaultFlowID)
		if flowRaw == "" {
			defaultFlow = pgtype.UUID{}
		} else {
			flowID, err := uuid.Parse(flowRaw)
			if err != nil {
				return CampaignGroupDTO{}, errValidation("invalid default_flow_id")
			}
			defaultFlow = domain.ToUUID(flowID)
		}
	}
	row, err := db.New(pool).UpdateCampaignGroup(ctx, db.UpdateCampaignGroupParams{
		ID:            domain.ToUUID(groupID),
		Name:          name,
		DefaultFlowID: defaultFlow,
	})
	if err != nil {
		return CampaignGroupDTO{}, mapCampaignGroupStoreError(err)
	}
	count, err := db.New(pool).CountCampaignsByGroup(ctx, db.CountCampaignsByGroupParams{
		CustomerID:      row.CustomerID,
		CampaignGroupID: row.ID,
	})
	if err != nil {
		return CampaignGroupDTO{}, err
	}
	return campaignGroupToDTO(row, count), nil
}

func DeleteCampaignGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) error {
	if pool == nil {
		return errServiceUnavailable()
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetCampaignGroup(ctx, domain.ToUUID(groupID)); err != nil {
			return mapCampaignGroupStoreError(err)
		}
		if err := q.ClearCampaignGroupMembers(ctx, domain.ToUUID(groupID)); err != nil {
			return err
		}
		if err := q.SoftDeleteCampaignGroup(ctx, domain.ToUUID(groupID)); err != nil {
			return err
		}
		return nil
	})
}

func AssignCampaignsToGroup(
	ctx context.Context,
	pool *pgxpool.Pool,
	groupID uuid.UUID,
	campaignIDs []uuid.UUID,
) (AssignCampaignGroupMembersResponse, error) {
	if pool == nil {
		return AssignCampaignGroupMembersResponse{}, errServiceUnavailable()
	}
	if len(campaignIDs) == 0 {
		return AssignCampaignGroupMembersResponse{}, errValidation("campaign_ids is required")
	}
	group, err := db.New(pool).GetCampaignGroup(ctx, domain.ToUUID(groupID))
	if err != nil {
		return AssignCampaignGroupMembersResponse{}, mapCampaignGroupStoreError(err)
	}
	customerID := uuid.UUID(group.CustomerID.Bytes)
	pgIDs := make([]pgtype.UUID, 0, len(campaignIDs))
	seen := make(map[uuid.UUID]struct{}, len(campaignIDs))
	for _, id := range campaignIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		pgIDs = append(pgIDs, domain.ToUUID(id))
	}
	if _, err := db.New(pool).AssignCampaignsToGroup(ctx, db.AssignCampaignsToGroupParams{
		CampaignGroupID: domain.ToUUID(groupID),
		CustomerID:      domain.ToUUID(customerID),
		CampaignIds:     pgIDs,
	}); err != nil {
		return AssignCampaignGroupMembersResponse{}, err
	}
	results := make([]CampaignGroupAssignResultRow, 0, len(campaignIDs))
	q := db.New(pool)
	for _, id := range campaignIDs {
		camp, err := q.GetCampaign(ctx, domain.ToUUID(id))
		if err != nil {
			results = append(results, CampaignGroupAssignResultRow{
				CampaignID: id.String(),
				ErrorCode:  "not_found",
			})
			continue
		}
		if !camp.CustomerID.Valid || uuid.UUID(camp.CustomerID.Bytes) != customerID {
			results = append(results, CampaignGroupAssignResultRow{
				CampaignID: id.String(),
				ErrorCode:  "customer_mismatch",
			})
			continue
		}
		if !camp.CampaignGroupID.Valid || uuid.UUID(camp.CampaignGroupID.Bytes) != groupID {
			results = append(results, CampaignGroupAssignResultRow{
				CampaignID: id.String(),
				ErrorCode:  "not_assigned",
			})
			continue
		}
		results = append(results, CampaignGroupAssignResultRow{CampaignID: id.String(), OK: true})
	}
	return AssignCampaignGroupMembersResponse{Results: results}, nil
}

func UnassignCampaignsFromGroup(
	ctx context.Context,
	pool *pgxpool.Pool,
	groupID uuid.UUID,
	campaignIDs []uuid.UUID,
) (AssignCampaignGroupMembersResponse, error) {
	if pool == nil {
		return AssignCampaignGroupMembersResponse{}, errServiceUnavailable()
	}
	if len(campaignIDs) == 0 {
		return AssignCampaignGroupMembersResponse{}, errValidation("campaign_ids is required")
	}
	group, err := db.New(pool).GetCampaignGroup(ctx, domain.ToUUID(groupID))
	if err != nil {
		return AssignCampaignGroupMembersResponse{}, mapCampaignGroupStoreError(err)
	}
	customerID := uuid.UUID(group.CustomerID.Bytes)
	pgIDs := make([]pgtype.UUID, 0, len(campaignIDs))
	for _, id := range campaignIDs {
		pgIDs = append(pgIDs, domain.ToUUID(id))
	}
	if _, err := db.New(pool).UnassignCampaignsFromGroup(ctx, db.UnassignCampaignsFromGroupParams{
		CustomerID:      domain.ToUUID(customerID),
		CampaignGroupID: domain.ToUUID(groupID),
		CampaignIds:     pgIDs,
	}); err != nil {
		return AssignCampaignGroupMembersResponse{}, err
	}
	results := make([]CampaignGroupAssignResultRow, 0, len(campaignIDs))
	q := db.New(pool)
	for _, id := range campaignIDs {
		camp, err := q.GetCampaign(ctx, domain.ToUUID(id))
		if err != nil {
			results = append(results, CampaignGroupAssignResultRow{
				CampaignID: id.String(),
				ErrorCode:  "not_found",
			})
			continue
		}
		if camp.CampaignGroupID.Valid {
			results = append(results, CampaignGroupAssignResultRow{
				CampaignID: id.String(),
				ErrorCode:  "still_assigned",
			})
			continue
		}
		results = append(results, CampaignGroupAssignResultRow{CampaignID: id.String(), OK: true})
	}
	return AssignCampaignGroupMembersResponse{Results: results}, nil
}

func ParseCampaignGroupIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty campaign_ids")
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		id, err := uuid.Parse(strings.TrimSpace(item))
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
