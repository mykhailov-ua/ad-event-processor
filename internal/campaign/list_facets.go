package campaign

import (
	"context"
	"net/http"
	"sort"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignListFacetOwner struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
}

type CampaignListFacetsResponse struct {
	Countries []string                 `json:"countries"`
	Owners    []CampaignListFacetOwner `json:"owners"`
}

func (h *CampaignsHTTPHandlers) listCampaignListFacets(w http.ResponseWriter, r *http.Request) {
	if h.PostgresPool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "postgres unavailable")
		return
	}

	var customerID uuid.UUID
	if custStr := r.URL.Query().Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = id
	}
	if h.ResolveCustomerID != nil {
		resolved, err := h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
		customerID = resolved
	}

	resp, err := QueryCampaignListFacets(r.Context(), h.PostgresPool, customerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}

func QueryCampaignListFacets(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (CampaignListFacetsResponse, error) {
	q := db.New(pool)
	var customerArg pgtype.UUID
	if customerID != uuid.Nil {
		customerArg = domain.ToUUID(customerID)
	}

	countries, err := q.ListCampaignTargetCountries(ctx, customerArg)
	if err != nil {
		return CampaignListFacetsResponse{}, err
	}
	if countries == nil {
		countries = []string{}
	}

	ownerIDs, err := q.ListCampaignListOwners(ctx, customerArg)
	if err != nil {
		return CampaignListFacetsResponse{}, err
	}

	owners, err := campaignListFacetOwnersWithEmail(ctx, pool, ownerIDs)
	if err != nil {
		return CampaignListFacetsResponse{}, err
	}

	return CampaignListFacetsResponse{
		Countries: countries,
		Owners:    owners,
	}, nil
}

func campaignListFacetOwnersWithEmail(ctx context.Context, pool *pgxpool.Pool, ownerIDs []string) ([]CampaignListFacetOwner, error) {
	if len(ownerIDs) == 0 {
		return []CampaignListFacetOwner{}, nil
	}

	emailByID, err := lookupCampaignOwnerEmails(ctx, pool, ownerIDs)
	if err != nil {
		return nil, err
	}

	owners := make([]CampaignListFacetOwner, 0, len(ownerIDs))
	for _, ownerID := range ownerIDs {
		owner := CampaignListFacetOwner{UserID: ownerID}
		if email := emailByID[ownerID]; email != "" {
			owner.Email = email
		}
		owners = append(owners, owner)
	}

	sort.Slice(owners, func(i, j int) bool {
		left := owners[i].Email
		if left == "" {
			left = owners[i].UserID
		}
		right := owners[j].Email
		if right == "" {
			right = owners[j].UserID
		}
		if left == right {
			return owners[i].UserID < owners[j].UserID
		}
		return left < right
	})

	return owners, nil
}

func lookupCampaignOwnerEmails(ctx context.Context, pool *pgxpool.Pool, ownerIDs []string) (map[string]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, COALESCE(NULLIF(btrim(email), ''), '') AS email
		FROM users
		WHERE id = ANY($1::uuid[])`, ownerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string, len(ownerIDs))
	for rows.Next() {
		var userID string
		var email string
		if err := rows.Scan(&userID, &email); err != nil {
			return nil, err
		}
		if email != "" {
			out[userID] = email
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
