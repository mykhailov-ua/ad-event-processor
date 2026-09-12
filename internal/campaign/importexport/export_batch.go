package importexport

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExportCampaignsBatchResult struct {
	Items  map[uuid.UUID]campaign.CampaignExportBundle
	Errors map[uuid.UUID]error
}

type flowExportCache struct {
	flow    *campaign.CampaignExportFlow
	landers map[uuid.UUID]campaign.CampaignExportLander
	offers  map[uuid.UUID]campaign.CampaignExportOffer
}

type bundleFlowRefs struct {
	campaignID uuid.UUID
	bundle     campaign.CampaignExportBundle
	landers    map[uuid.UUID]campaign.CampaignExportLander
	offers     map[uuid.UUID]campaign.CampaignExportOffer
}

func ExportCampaignsBatch(ctx context.Context, host campaign.ImportExportHost, ids []uuid.UUID) ExportCampaignsBatchResult {
	out := ExportCampaignsBatchResult{
		Items:  make(map[uuid.UUID]campaign.CampaignExportBundle),
		Errors: make(map[uuid.UUID]error),
	}
	if host == nil || host.Pool() == nil {
		for _, id := range ids {
			out.Errors[id] = fmt.Errorf("service unavailable")
		}
		return out
	}
	if len(ids) == 0 {
		return out
	}

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = domain.ToUUID(id)
	}

	q := db.New(host.Pool())
	rows, err := q.ListCampaignsByIDs(ctx, pgIDs)
	if err != nil {
		for _, id := range ids {
			out.Errors[id] = err
		}
		return out
	}
	rowByID := make(map[uuid.UUID]db.Campaign, len(rows))
	for i := range rows {
		rowByID[uuid.UUID(rows[i].ID.Bytes)] = rows[i]
	}

	exportedAt := time.Now().UTC().Format(time.RFC3339)
	needPostback := make([]pgtype.UUID, 0, len(ids))
	needOutbound := make([]pgtype.UUID, 0, len(ids))
	needMappings := make([]pgtype.UUID, 0, len(ids))
	schemaIDs := make([]uuid.UUID, 0, 8)
	flowIDList := make([]uuid.UUID, 0, 8)
	type pendingExport struct {
		id   uuid.UUID
		row  db.Campaign
		flow uuid.UUID
	}
	pending := make([]pendingExport, 0, len(ids))

	for _, id := range ids {
		row, ok := rowByID[id]
		if !ok {
			out.Errors[id] = campaign.ErrCampaignNotFound
			continue
		}
		if err := host.AssertMediaBuyerCampaignAccess(ctx, row); err != nil {
			out.Errors[id] = err
			continue
		}
		if row.DeletedAt.Valid {
			out.Errors[id] = campaign.ErrCampaignNotFound
			continue
		}
		var flowID uuid.UUID
		if row.FlowID.Valid {
			flowID = uuid.UUID(row.FlowID.Bytes)
			flowIDList = append(flowIDList, flowID)
		}
		if row.IntegrationSchemaID.Valid {
			schemaIDs = append(schemaIDs, uuid.UUID(row.IntegrationSchemaID.Bytes))
		}
		if row.StatusIntegrationSchemaID.Valid {
			schemaIDs = append(schemaIDs, uuid.UUID(row.StatusIntegrationSchemaID.Bytes))
		}
		pgID := domain.ToUUID(id)
		needPostback = append(needPostback, pgID)
		needOutbound = append(needOutbound, pgID)
		needMappings = append(needMappings, pgID)
		pending = append(pending, pendingExport{id: id, row: row, flow: flowID})
	}

	postbackByCampaign := map[uuid.UUID]db.PostbackConfig{}
	if len(needPostback) > 0 {
		pbRows, pbErr := q.ListPostbackConfigsByCampaignIDs(ctx, needPostback)
		if pbErr != nil {
			for _, pe := range pending {
				out.Errors[pe.id] = pbErr
			}
			return out
		}
		for i := range pbRows {
			cid := uuid.UUID(pbRows[i].CampaignID.Bytes)
			postbackByCampaign[cid] = pbRows[i]
		}
	}

	outboundByCampaign := map[uuid.UUID][]db.CampaignOutboundPostback{}
	if len(needOutbound) > 0 {
		outboundRows, outboundErr := q.ListOutboundPostbacksByCampaignIDs(ctx, needOutbound)
		if outboundErr != nil {
			for _, pe := range pending {
				out.Errors[pe.id] = outboundErr
			}
			return out
		}
		for i := range outboundRows {
			row := db.CampaignOutboundPostbackFromListIDs(outboundRows[i])
			cid := uuid.UUID(row.CampaignID.Bytes)
			outboundByCampaign[cid] = append(outboundByCampaign[cid], row)
		}
	}

	mappingsByCampaign := map[uuid.UUID][]db.CampaignConversionMapping{}
	if len(needMappings) > 0 {
		mapRows, mapErr := q.ListConversionMappingsByCampaignIDs(ctx, needMappings)
		if mapErr != nil {
			for _, pe := range pending {
				out.Errors[pe.id] = mapErr
			}
			return out
		}
		for i := range mapRows {
			cid := uuid.UUID(mapRows[i].CampaignID.Bytes)
			mappingsByCampaign[cid] = append(mappingsByCampaign[cid], mapRows[i])
		}
	}

	schemaNames, schemaErr := integrationSchemaNamesByIDs(ctx, host.Pool(), schemaIDs)
	if schemaErr != nil {
		for _, pe := range pending {
			out.Errors[pe.id] = schemaErr
		}
		return out
	}

	flowCache := make(map[uuid.UUID]flowExportCache, len(flowIDList))
	for _, flowID := range uniqueUUIDs(flowIDList) {
		flow, flowErr := host.GetFlow(ctx, flowID)
		if flowErr != nil {
			for _, pe := range pending {
				if pe.flow == flowID {
					out.Errors[pe.id] = flowErr
				}
			}
			continue
		}
		exportFlow, landerRefs, offerRefs, convErr := exportFlowBundle(flow)
		if convErr != nil {
			for _, pe := range pending {
				if pe.flow == flowID {
					out.Errors[pe.id] = convErr
				}
			}
			continue
		}
		flowCache[flowID] = flowExportCache{
			flow:    exportFlow,
			landers: landerRefs,
			offers:  offerRefs,
		}
	}

	flowRefBundles := make([]bundleFlowRefs, 0, len(pending))

	for _, pe := range pending {
		if _, failed := out.Errors[pe.id]; failed {
			continue
		}
		row := pe.row
		bundle := campaign.CampaignExportBundle{
			ExportVersion: CampaignExportVersion,
			ExportedAt:    exportedAt,
			Campaign:      campaignRowToExport(row),
		}
		if row.IntegrationSchemaID.Valid {
			bundle.IntegrationSchemaName = schemaNames[uuid.UUID(row.IntegrationSchemaID.Bytes)]
		}
		if row.StatusIntegrationSchemaID.Valid {
			bundle.StatusIntegrationSchemaName = schemaNames[uuid.UUID(row.StatusIntegrationSchemaID.Bytes)]
		}
		if pb, ok := postbackByCampaign[pe.id]; ok {
			bundle.PostbackConfig = &campaign.CampaignExportPostback{
				Provider:      pb.Provider,
				URLTemplate:   pb.UrlTemplate,
				TargetEvent:   pb.TargetEvent,
				TestEventCode: pb.TestEventCode,
			}
		}
		for _, mapping := range mappingsByCampaign[pe.id] {
			bundle.ConversionMappings = append(bundle.ConversionMappings, campaign.ConversionMappingToDTO(&mapping))
		}
		if outboundRows, ok := outboundByCampaign[pe.id]; ok && len(outboundRows) > 0 {
			bundle.OutboundPostbacks = campaign.ExportOutboundPostbacksFromRows(outboundRows)
		}
		if pe.flow != uuid.Nil {
			cache, ok := flowCache[pe.flow]
			if !ok {
				out.Errors[pe.id] = fmt.Errorf("flow export cache miss")
				continue
			}
			bundle.Flow = cloneExportFlow(cache.flow)
			flowRefBundles = append(flowRefBundles, bundleFlowRefs{
				campaignID: pe.id,
				bundle:     bundle,
				landers:    cloneLanderRefs(cache.landers),
				offers:     cloneOfferRefs(cache.offers),
			})
			continue
		}
		if err := finalizeExportBundle(&bundle); err != nil {
			out.Errors[pe.id] = err
			continue
		}
		out.Items[pe.id] = bundle
	}

	if len(flowRefBundles) > 0 {
		if err := enrichExportFlowAssetsBatch(ctx, host.Pool(), flowRefBundles); err != nil {
			for i := range flowRefBundles {
				out.Errors[flowRefBundles[i].campaignID] = err
			}
		} else {
			for i := range flowRefBundles {
				cid := flowRefBundles[i].campaignID
				if _, failed := out.Errors[cid]; failed {
					continue
				}
				bundle := flowRefBundles[i].bundle
				if err := finalizeExportBundle(&bundle); err != nil {
					out.Errors[cid] = err
					continue
				}
				out.Items[cid] = bundle
			}
		}
	}

	return out
}

func finalizeExportBundle(bundle *campaign.CampaignExportBundle) error {
	raw, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	if len(raw) > MaxCampaignImportBytes {
		return fmt.Errorf("export bundle exceeds %d bytes", MaxCampaignImportBytes)
	}
	return nil
}

func integrationSchemaNamesByIDs(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	unique := uniqueUUIDs(ids)
	if len(unique) == 0 || pool == nil {
		return map[uuid.UUID]string{}, nil
	}
	rows, err := pool.Query(ctx, `SELECT id, name FROM integration_schemas WHERE id = ANY($1)`, unique)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]string, len(unique))
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

func enrichExportFlowAssetsBatch(ctx context.Context, pool *pgxpool.Pool, bundles []bundleFlowRefs) error {
	if pool == nil || len(bundles) == 0 {
		return nil
	}
	landerIDs := make(map[uuid.UUID]struct{})
	offerIDs := make(map[uuid.UUID]struct{})
	for i := range bundles {
		for id := range bundles[i].landers {
			landerIDs[id] = struct{}{}
		}
		for id := range bundles[i].offers {
			offerIDs[id] = struct{}{}
		}
	}
	landerNames := make(map[uuid.UUID]struct {
		name string
		url  string
	}, len(landerIDs))
	if len(landerIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(landerIDs))
		for id := range landerIDs {
			ids = append(ids, id)
		}
		rows, err := pool.Query(ctx, `SELECT id, name, COALESCE(url, '') FROM landers WHERE id = ANY($1)`, ids)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			var name, url string
			if err := rows.Scan(&id, &name, &url); err != nil {
				return err
			}
			landerNames[id] = struct {
				name string
				url  string
			}{name: name, url: url}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}
	offerNames := make(map[uuid.UUID]struct {
		name string
		url  string
	}, len(offerIDs))
	if len(offerIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(offerIDs))
		for id := range offerIDs {
			ids = append(ids, id)
		}
		rows, err := pool.Query(ctx, `SELECT id, name, url FROM offers WHERE id = ANY($1)`, ids)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			var name, url string
			if err := rows.Scan(&id, &name, &url); err != nil {
				return err
			}
			offerNames[id] = struct {
				name string
				url  string
			}{name: name, url: url}
		}
		if err := rows.Err(); err != nil {
			return err
		}
	}
	for i := range bundles {
		bundle := &bundles[i].bundle
		for id, ref := range bundles[i].landers {
			if row, ok := landerNames[id]; ok {
				ref.Name = row.name
				ref.URL = row.url
			}
			bundle.Landers = append(bundle.Landers, ref)
		}
		for id, ref := range bundles[i].offers {
			if row, ok := offerNames[id]; ok {
				ref.Name = row.name
				ref.URL = row.url
			}
			bundle.Offers = append(bundle.Offers, ref)
		}
	}
	return nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func cloneExportFlow(flow *campaign.CampaignExportFlow) *campaign.CampaignExportFlow {
	if flow == nil {
		return nil
	}
	out := &campaign.CampaignExportFlow{Name: flow.Name, Paths: make([]campaign.CampaignExportFlowPath, len(flow.Paths))}
	copy(out.Paths, flow.Paths)
	return out
}

func cloneLanderRefs(in map[uuid.UUID]campaign.CampaignExportLander) map[uuid.UUID]campaign.CampaignExportLander {
	out := make(map[uuid.UUID]campaign.CampaignExportLander, len(in))
	for id, ref := range in {
		out[id] = ref
	}
	return out
}

func cloneOfferRefs(in map[uuid.UUID]campaign.CampaignExportOffer) map[uuid.UUID]campaign.CampaignExportOffer {
	out := make(map[uuid.UUID]campaign.CampaignExportOffer, len(in))
	for id, ref := range in {
		out[id] = ref
	}
	return out
}
