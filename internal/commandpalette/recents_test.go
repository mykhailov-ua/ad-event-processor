package commandpalette

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRecentsQuerier struct {
	listRows       []db.ListCommandPaletteRecentsRow
	listCustomerID pgtype.UUID
	listUserID     pgtype.UUID
	upsertCalls    int
	pruneKeepMax   int32
}

func (f *fakeRecentsQuerier) ListCommandPaletteRecents(_ context.Context, arg db.ListCommandPaletteRecentsParams) ([]db.ListCommandPaletteRecentsRow, error) {
	f.listCustomerID = arg.CustomerID
	f.listUserID = arg.UserID
	return f.listRows, nil
}

func (f *fakeRecentsQuerier) UpsertCommandPaletteRecent(context.Context, db.UpsertCommandPaletteRecentParams) error {
	f.upsertCalls++
	return nil
}

func (f *fakeRecentsQuerier) PruneCommandPaletteRecents(_ context.Context, arg db.PruneCommandPaletteRecentsParams) error {
	f.pruneKeepMax = arg.KeepMax
	return nil
}

func TestRecentsStore_ListRecents_mapsRows(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	fake := &fakeRecentsQuerier{
		listRows: []db.ListCommandPaletteRecentsRow{{
			ItemID: "camp-1",
			Kind:   "campaign",
			Label:  "Camp Alpha",
			Href:   "/campaigns/camp-1",
			Meta:   pgtype.Text{String: "meta", Valid: true},
			Group:  pgtype.Text{String: "campaigns", Valid: true},
		}},
	}
	st := &RecentsStore{q: fake}
	items, err := st.ListRecents(t.Context(), custID, userID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "camp-1", items[0].ID)
	assert.Equal(t, "campaign", items[0].Kind)
	assert.Equal(t, "Camp Alpha", items[0].Label)
	assert.Equal(t, "/campaigns/camp-1", items[0].Href)
	assert.Equal(t, "meta", items[0].Meta)
	assert.Equal(t, "campaigns", items[0].Group)
}

func TestRecentsStore_RecordRecent_upsertsAndPrunes(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	fake := &fakeRecentsQuerier{}
	st := &RecentsStore{q: fake}
	err := st.RecordRecent(t.Context(), custID, userID, ItemDTO{
		ID:    "route-integrations",
		Kind:  "route",
		Label: "Integrations",
		Href:  "/integrations",
		Group: "settings",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, fake.upsertCalls)
	assert.Equal(t, int32(maxRecentsPerUser), fake.pruneKeepMax)
}

func TestHTTPHandlers_listRecents_foreignCustomer_holdout(t *testing.T) {
	t.Parallel()
	custA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	custB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	fake := &fakeRecentsQuerier{
		listRows: []db.ListCommandPaletteRecentsRow{{
			ItemID: "foreign-camp",
			Kind:   "campaign",
			Label:  "Foreign Camp",
			Href:   "/campaigns/foreign-camp",
		}},
	}
	h := &HTTPHandlers{
		Recents:           &RecentsStore{q: fake},
		ResolveCustomerID: boundCustomerResolver(custA),
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/recents?customer_id="+custB.String(), http.NoBody)
	req = req.WithContext(authz.WithAuthenticatedUser(req.Context(), authz.AuthenticatedUser{
		UserID:     userID,
		Role:       authz.RoleBuyer,
		CustomerID: custA,
	}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Equal(t, 0, fake.upsertCalls)
	assert.False(t, fake.listCustomerID.Valid)
}

func TestHTTPHandlers_listRecents_returnsItems(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	fake := &fakeRecentsQuerier{
		listRows: []db.ListCommandPaletteRecentsRow{{
			ItemID: "camp-1",
			Kind:   "campaign",
			Label:  "Camp Alpha",
			Href:   "/campaigns/camp-1",
		}},
	}
	h := &HTTPHandlers{
		Recents:           &RecentsStore{q: fake},
		ResolveCustomerID: boundCustomerResolver(custID),
		RequireAnyPermission: func(_ []string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/recents?customer_id="+custID.String(), http.NoBody)
	req = req.WithContext(authz.WithAuthenticatedUser(req.Context(), authz.AuthenticatedUser{
		UserID:     userID,
		Role:       authz.RoleBuyer,
		CustomerID: custID,
	}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp RecentsResponse
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "camp-1", resp.Items[0].ID)
}

func boundCustomerResolver(boundCustomerID uuid.UUID) func(*http.Request, *uuid.UUID) (uuid.UUID, error) {
	return func(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
		u, ok := authz.GetUser(r.Context())
		if !ok {
			return uuid.Nil, errForbidden
		}
		if u.HasBoundCustomer() {
			if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
				return uuid.Nil, errForbidden
			}
			return u.CustomerID, nil
		}
		if bodyCustomerID == nil || *bodyCustomerID == uuid.Nil {
			return uuid.Nil, errCustomerIDRequired
		}
		return *bodyCustomerID, nil
	}
}
