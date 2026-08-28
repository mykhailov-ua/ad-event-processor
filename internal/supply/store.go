package supply

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	host Host
}

func NewStore(pool *pgxpool.Pool, host Host) *Store {
	return &Store{pool: pool, host: host}
}

const (
	sellersJSONCacheTTL  = 60 * time.Second
	sellersJSONVersion   = "1.0"
	supplySettingOwner   = "supply_owner_domain"
	supplySettingManager = "supply_manager_domain"
	supplySettingContact = "supply_contact_email"
)

type sellersJSONCacheEntry struct {
	body    []byte
	expires time.Time
}

type sellersJSONCache struct {
	mu sync.RWMutex
	v  sellersJSONCacheEntry
}

var sellersCache sellersJSONCache

func (st *Store) enqueueSupplyFilesUpdate(ctx context.Context, q db.Querier, trigger string) error {
	InvalidateSellersJSONCache()
	return st.host.EnqueueSupplyFilesUpdate(ctx, q, trigger)
}

func InvalidateSellersJSONCache() {
	sellersCache.mu.Lock()
	sellersCache.v = sellersJSONCacheEntry{}
	sellersCache.mu.Unlock()
}

func (st *Store) ListSellers(ctx context.Context) ([]SellerDTO, error) {
	rows, err := db.New(st.pool).ListSellers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SellerDTO, len(rows))
	for i, r := range rows {
		out[i] = SellerDTO{
			ID:             r.ID,
			SellerID:       r.SellerID,
			Domain:         r.Domain,
			SellerType:     r.SellerType,
			Name:           r.Name,
			IsConfidential: r.IsConfidential,
			CreatedAt:      r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (st *Store) GetSeller(ctx context.Context, id int64) (SellerDTO, error) {
	row, err := db.New(st.pool).GetSeller(ctx, id)
	if err != nil {
		return SellerDTO{}, ErrSellerNotFound
	}
	return SellerDTO{
		ID:             row.ID,
		SellerID:       row.SellerID,
		Domain:         row.Domain,
		SellerType:     row.SellerType,
		Name:           row.Name,
		IsConfidential: row.IsConfidential,
		CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (st *Store) CreateSeller(ctx context.Context, spec SellerCreateSpec) (SellerDTO, error) {
	sellerType, err := NormalizeSellerType(spec.SellerType)
	if err != nil {
		return SellerDTO{}, err
	}
	if strings.TrimSpace(spec.SellerID) == "" || strings.TrimSpace(spec.Domain) == "" {
		return SellerDTO{}, st.host.ErrValidation("seller_id and domain are required")
	}

	var out SellerDTO
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.CreateSeller(ctx, db.CreateSellerParams{
			SellerID:       strings.TrimSpace(spec.SellerID),
			Domain:         strings.TrimSpace(spec.Domain),
			SellerType:     sellerType,
			Name:           strings.TrimSpace(spec.Name),
			IsConfidential: spec.IsConfidential,
		})
		if err != nil {
			return err
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "CREATE_SELLER", "supply", nil, auditSellerCreateChange{
			SellerID: row.SellerID,
			Domain:   row.Domain,
		}, nil)

		if err := st.enqueueSupplyFilesUpdate(ctx, q, "create_seller"); err != nil {
			return err
		}
		out = SellerDTO{
			ID:             row.ID,
			SellerID:       row.SellerID,
			Domain:         row.Domain,
			SellerType:     row.SellerType,
			Name:           row.Name,
			IsConfidential: row.IsConfidential,
			CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (st *Store) UpdateSeller(ctx context.Context, id int64, spec SellerUpdateSpec) (SellerDTO, error) {
	sellerType, err := NormalizeSellerType(spec.SellerType)
	if err != nil {
		return SellerDTO{}, err
	}
	if strings.TrimSpace(spec.SellerID) == "" || strings.TrimSpace(spec.Domain) == "" {
		return SellerDTO{}, st.host.ErrValidation("seller_id and domain are required")
	}

	var out SellerDTO
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.UpdateSeller(ctx, db.UpdateSellerParams{
			ID:             id,
			SellerID:       strings.TrimSpace(spec.SellerID),
			Domain:         strings.TrimSpace(spec.Domain),
			SellerType:     sellerType,
			Name:           strings.TrimSpace(spec.Name),
			IsConfidential: spec.IsConfidential,
		})
		if err != nil {
			return ErrSellerNotFound
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "UPDATE_SELLER", "supply", nil, auditSellerUpdateChange{
			ID:       id,
			SellerID: row.SellerID,
		}, nil)

		if err := st.enqueueSupplyFilesUpdate(ctx, q, "update_seller"); err != nil {
			return err
		}
		out = SellerDTO{
			ID:             row.ID,
			SellerID:       row.SellerID,
			Domain:         row.Domain,
			SellerType:     row.SellerType,
			Name:           row.Name,
			IsConfidential: row.IsConfidential,
			CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (st *Store) DeleteSeller(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetSeller(ctx, id); err != nil {
			return ErrSellerNotFound
		}
		if err := q.DeleteSeller(ctx, id); err != nil {
			return err
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "DELETE_SELLER", "supply", nil, auditIDChange{ID: id}, nil)
		return st.enqueueSupplyFilesUpdate(ctx, q, "delete_seller")
	})
}

func (st *Store) ListAdsTxtEntries(ctx context.Context) ([]AdsTxtEntryDTO, error) {
	rows, err := db.New(st.pool).ListAdsTxtEntries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdsTxtEntryDTO, len(rows))
	for i, r := range rows {
		out[i] = AdsTxtEntryDTO{
			ID:                 r.ID,
			Domain:             r.Domain,
			PublisherAccountID: r.PublisherAccountID,
			Relationship:       r.Relationship,
			CertAuthorityID:    r.CertAuthorityID,
			SortOrder:          r.SortOrder,
			CreatedAt:          r.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          r.UpdatedAt.Time.Format(time.RFC3339),
		}
	}
	return out, nil
}

func (st *Store) GetAdsTxtEntry(ctx context.Context, id int64) (AdsTxtEntryDTO, error) {
	row, err := db.New(st.pool).GetAdsTxtEntry(ctx, id)
	if err != nil {
		return AdsTxtEntryDTO{}, ErrAdsTxtEntryNotFound
	}
	return AdsTxtEntryDTO{
		ID:                 row.ID,
		Domain:             row.Domain,
		PublisherAccountID: row.PublisherAccountID,
		Relationship:       row.Relationship,
		CertAuthorityID:    row.CertAuthorityID,
		SortOrder:          row.SortOrder,
		CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
	}, nil
}

func (st *Store) CreateAdsTxtEntry(ctx context.Context, spec AdsTxtEntryCreateSpec) (AdsTxtEntryDTO, error) {
	rel, err := NormalizeRelationship(spec.Relationship)
	if err != nil {
		return AdsTxtEntryDTO{}, err
	}
	if strings.TrimSpace(spec.Domain) == "" || strings.TrimSpace(spec.PublisherAccountID) == "" {
		return AdsTxtEntryDTO{}, st.host.ErrValidation("domain and publisher_account_id are required")
	}

	var out AdsTxtEntryDTO
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.CreateAdsTxtEntry(ctx, db.CreateAdsTxtEntryParams{
			Domain:             strings.TrimSpace(spec.Domain),
			PublisherAccountID: strings.TrimSpace(spec.PublisherAccountID),
			Relationship:       rel,
			CertAuthorityID:    strings.TrimSpace(spec.CertAuthorityID),
			SortOrder:          spec.SortOrder,
		})
		if err != nil {
			return err
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "CREATE_ADS_TXT", "supply", nil, auditAdsTxtDomainChange{
			Domain: spec.Domain,
		}, nil)

		if err := st.enqueueSupplyFilesUpdate(ctx, q, "create_ads_txt"); err != nil {
			return err
		}
		out = AdsTxtEntryDTO{
			ID:                 row.ID,
			Domain:             row.Domain,
			PublisherAccountID: row.PublisherAccountID,
			Relationship:       row.Relationship,
			CertAuthorityID:    row.CertAuthorityID,
			SortOrder:          row.SortOrder,
			CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (st *Store) UpdateAdsTxtEntry(ctx context.Context, id int64, spec AdsTxtEntryUpdateSpec) (AdsTxtEntryDTO, error) {
	rel, err := NormalizeRelationship(spec.Relationship)
	if err != nil {
		return AdsTxtEntryDTO{}, err
	}
	if strings.TrimSpace(spec.Domain) == "" || strings.TrimSpace(spec.PublisherAccountID) == "" {
		return AdsTxtEntryDTO{}, st.host.ErrValidation("domain and publisher_account_id are required")
	}

	var out AdsTxtEntryDTO
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.UpdateAdsTxtEntry(ctx, db.UpdateAdsTxtEntryParams{
			ID:                 id,
			Domain:             strings.TrimSpace(spec.Domain),
			PublisherAccountID: strings.TrimSpace(spec.PublisherAccountID),
			Relationship:       rel,
			CertAuthorityID:    strings.TrimSpace(spec.CertAuthorityID),
			SortOrder:          spec.SortOrder,
		})
		if err != nil {
			return ErrAdsTxtEntryNotFound
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "UPDATE_ADS_TXT", "supply", nil, auditIDChange{ID: id}, nil)

		if err := st.enqueueSupplyFilesUpdate(ctx, q, "update_ads_txt"); err != nil {
			return err
		}
		out = AdsTxtEntryDTO{
			ID:                 row.ID,
			Domain:             row.Domain,
			PublisherAccountID: row.PublisherAccountID,
			Relationship:       row.Relationship,
			CertAuthorityID:    row.CertAuthorityID,
			SortOrder:          row.SortOrder,
			CreatedAt:          row.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:          row.UpdatedAt.Time.Format(time.RFC3339),
		}
		return nil
	})
	return out, err
}

func (st *Store) DeleteAdsTxtEntry(ctx context.Context, id int64) error {
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.GetAdsTxtEntry(ctx, id); err != nil {
			return ErrAdsTxtEntryNotFound
		}
		if err := q.DeleteAdsTxtEntry(ctx, id); err != nil {
			return err
		}

		var uid = st.host.ActorUserID(ctx)
		st.host.AuditLog(ctx, q, uid, "DELETE_ADS_TXT", "supply", nil, auditIDChange{ID: id}, nil)
		return st.enqueueSupplyFilesUpdate(ctx, q, "delete_ads_txt")
	})
}
type iabSellersJSON struct {
	ContactEmail string          `json:"contact_email,omitempty"`
	Version      string          `json:"version"`
	Sellers      []iabSellerJSON `json:"sellers"`
}

type iabSellerJSON struct {
	SellerID       string `json:"seller_id"`
	Name           string `json:"name,omitempty"`
	Domain         string `json:"domain"`
	SellerType     string `json:"seller_type"`
	IsConfidential int    `json:"is_confidential,omitempty"`
}

func validateSellersJSON(doc iabSellersJSON) error {
	if strings.TrimSpace(doc.Version) == "" {
		return fmt.Errorf("%w: version required", ErrSellersJSONInvalid)
	}
	if doc.Sellers == nil {
		return fmt.Errorf("%w: sellers array required", ErrSellersJSONInvalid)
	}
	for i, s := range doc.Sellers {
		if strings.TrimSpace(s.SellerID) == "" || strings.TrimSpace(s.Domain) == "" {
			return fmt.Errorf("%w: seller %d missing seller_id or domain", ErrSellersJSONInvalid, i)
		}
		if _, err := NormalizeSellerType(s.SellerType); err != nil {
			return fmt.Errorf("%w: seller %d invalid seller_type", ErrSellersJSONInvalid, i)
		}
	}
	return nil
}

func (st *Store) BuildSellersJSON(ctx context.Context) ([]byte, error) {
	q := db.New(st.pool)
	rows, err := q.ListSellers(ctx)
	if err != nil {
		return nil, err
	}

	settings, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	settingsMap := make(map[string]string, len(settings))
	for _, r := range settings {
		settingsMap[r.Key] = r.Value
	}

	doc := iabSellersJSON{
		Version: sellersJSONVersion,
		Sellers: make([]iabSellerJSON, 0, len(rows)),
	}
	if email := strings.TrimSpace(settingsMap[supplySettingContact]); email != "" {
		doc.ContactEmail = email
	}

	for _, row := range rows {
		entry := iabSellerJSON{
			SellerID:   row.SellerID,
			Domain:     row.Domain,
			SellerType: row.SellerType,
			Name:       row.Name,
		}
		if row.IsConfidential {
			entry.IsConfidential = 1
		}
		doc.Sellers = append(doc.Sellers, entry)
	}

	if err := validateSellersJSON(doc); err != nil {
		return nil, err
	}
	return coldpath.MarshalJSON(doc)
}

func (st *Store) GetSellersJSON(ctx context.Context) ([]byte, error) {
	now := time.Now()
	sellersCache.mu.RLock()
	if len(sellersCache.v.body) > 0 && now.Before(sellersCache.v.expires) {
		body := sellersCache.v.body
		sellersCache.mu.RUnlock()
		return body, nil
	}
	sellersCache.mu.RUnlock()

	body, err := st.BuildSellersJSON(ctx)
	if err != nil {
		return nil, err
	}

	sellersCache.mu.Lock()
	sellersCache.v = sellersJSONCacheEntry{body: body, expires: now.Add(sellersJSONCacheTTL)}
	sellersCache.mu.Unlock()
	return body, nil
}

func (st *Store) BuildAdsTxt(ctx context.Context) (string, error) {
	q := db.New(st.pool)
	rows, err := q.ListAdsTxtEntries(ctx)
	if err != nil {
		return "", err
	}
	settings, err := q.GetAllSystemSettings(ctx)
	if err != nil {
		return "", err
	}
	settingsMap := make(map[string]string, len(settings))
	for _, r := range settings {
		settingsMap[r.Key] = r.Value
	}

	var b strings.Builder
	if owner := strings.TrimSpace(settingsMap[supplySettingOwner]); owner != "" {
		b.WriteString("OWNERDOMAIN=")
		b.WriteString(owner)
		b.WriteByte('\n')
	}
	if manager := strings.TrimSpace(settingsMap[supplySettingManager]); manager != "" {
		b.WriteString("MANAGERDOMAIN=")
		b.WriteString(manager)
		b.WriteByte('\n')
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}

	for _, row := range rows {
		b.WriteString(row.Domain)
		b.WriteString(", ")
		b.WriteString(row.PublisherAccountID)
		b.WriteString(", ")
		b.WriteString(row.Relationship)
		if cert := strings.TrimSpace(row.CertAuthorityID); cert != "" {
			b.WriteString(", ")
			b.WriteString(cert)
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (st *Store) SupplyExportPath() string {
	return st.host.SupplyExportPath()
}
