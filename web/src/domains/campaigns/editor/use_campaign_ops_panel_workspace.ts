// campaign ops tab: stats/margin/events/mappings lanes; integration debugger lives under /integrations/debugger.
import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  blockCampaignPlacement,
  getCampaignMargin,
  getCampaignStats,
  getPlacementBlockSuggestions,
  listCampaignConversionMappings,
  listCampaignEvents,
  replaceCampaignConversionMappings,
  type CampaignListMetrics,
} from '@/api/campaigns_api';
import { applyIntegrationSchema } from '@/api/integrations_api';
import type {
  Campaign,
  CampaignEventListResponse,
  CampaignMargin,
  CampaignStats,
  CampaignStatsQuery,
  ConversionMapping,
  ConversionMappingListResponse,
  PlacementBlockSuggestion,
} from '@/api/types';
import { requireInteger, requireNonEmpty, validationError } from '@/lib/admin_validation_error';
import {
  buildCampaignStatsCacheKey,
  campaignStatsFromListMetrics,
  readCachedCampaignStats,
  readCachedCampaignStatsForCampaign,
  writeCachedCampaignStats,
} from '@/domains/campaigns/list/campaign_list_stats_cache';

type MappingDraft = {
  inbound_status: string;
  goal_name: string;
  payout_micro: string;
};

function mappingToDraft(mapping: ConversionMapping): MappingDraft {
  return {
    inbound_status: mapping.inbound_status ?? '',
    goal_name: mapping.goal_name ?? '',
    payout_micro: mapping.payout_micro != null ? String(mapping.payout_micro) : '',
  };
}

function draftsToMappings(drafts: MappingDraft[]): ConversionMapping[] {
  return drafts.map((draft) => {
    const mapping: ConversionMapping = {
      inbound_status: draft.inbound_status.trim(),
      goal_name: draft.goal_name.trim(),
    };
    const payout = draft.payout_micro.trim();
    if (payout) {
      const payoutMicro = requireInteger(payout, 'Payout micro', { min: 0, field: 'payout_micro' });
      if (!payoutMicro.ok) {
        throw payoutMicro.error;
      }
      mapping.payout_micro = payoutMicro.value;
    }
    return mapping;
  });
}

type UseCampaignOpsPanelWorkspaceArgs = {
  campaignId: string;
  campaign?: Campaign;
  listMetrics?: CampaignListMetrics;
  listMargin?: CampaignMargin;
  statsQuery?: CampaignStatsQuery;
};

export function useCampaignOpsPanelWorkspace({
  campaignId,
  campaign,
  listMetrics,
  listMargin,
  statsQuery,
}: UseCampaignOpsPanelWorkspaceArgs) {
  const [draftPlacementId, setDraftPlacementId] = useState('');
  const [blocking, setBlocking] = useState(false);
  const [loadingKey, setLoadingKey] = useState<string | undefined>();
  const [stats, setStats] = useState<CampaignStats | undefined>();
  const [events, setEvents] = useState<CampaignEventListResponse | undefined>();
  const [margin, setMargin] = useState<CampaignMargin | undefined>();
  const [mappings, setMappings] = useState<ConversionMappingListResponse | undefined>();
  const [mappingDrafts, setMappingDrafts] = useState<MappingDraft[]>([]);
  const [suggestions, setSuggestions] = useState<PlacementBlockSuggestion[]>([]);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [statsError, setStatsError] = useState<Error | undefined>();
  const [savingMappings, setSavingMappings] = useState(false);
  const [mappingSaveSuccess, setMappingSaveSuccess] = useState(false);
  const [syncingPreset, setSyncingPreset] = useState(false);
  const [syncPresetMessage, setSyncPresetMessage] = useState<string | undefined>();

  const resolvedStatsQuery = useMemo(
    () => statsQuery ?? {},
    [statsQuery?.from, statsQuery?.granularity, statsQuery?.to]
  );
  const statsCacheRevision = useMemo(() => `editor:${campaignId}`, [campaignId]);

  useEffect(() => {
    setStats(undefined);
    setEvents(undefined);
    setMargin(undefined);
    setMappings(undefined);
    setMappingDrafts([]);
    setSuggestions([]);
    setActionError(undefined);
    setStatsError(undefined);
    setMappingSaveSuccess(false);
    setSyncPresetMessage(undefined);
  }, [campaignId]);

  useEffect(() => {
    if (!mappings?.mappings) {
      return;
    }
    setMappingDrafts(mappings.mappings.map(mappingToDraft));
  }, [mappings]);

  const runAction = useCallback(async (key: string, action: () => Promise<void>) => {
    setLoadingKey(key);
    setActionError(undefined);
    try {
      await action();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoadingKey(undefined);
    }
  }, []);

  const onLoadStats = useCallback(() => {
    void runAction('stats', async () => {
      setStatsError(undefined);
      const editorCacheKey = buildCampaignStatsCacheKey(
        campaignId,
        resolvedStatsQuery,
        statsCacheRevision
      );

      if (!stats) {
        const cached =
          readCachedCampaignStats(editorCacheKey) ??
          readCachedCampaignStatsForCampaign(campaignId, resolvedStatsQuery);
        if (cached) {
          setStats(cached);
          return;
        }
      }

      const seeded =
        !stats && listMetrics
          ? campaignStatsFromListMetrics(campaignId, listMetrics, resolvedStatsQuery)
          : undefined;
      if (seeded) {
        setStats(seeded);
      }

      try {
        const fetched = await getCampaignStats(campaignId, resolvedStatsQuery);
        writeCachedCampaignStats(editorCacheKey, fetched);
        setStats(fetched);
      } catch (err: unknown) {
        setStatsError(err instanceof Error ? err : new Error(String(err)));
      }
    });
  }, [campaignId, listMetrics, resolvedStatsQuery, runAction, stats, statsCacheRevision]);

  useEffect(() => {
    onLoadStats();
  }, [campaignId, onLoadStats]);

  const onLoadEvents = useCallback(() => {
    void runAction('events', async () => {
      setEvents(await listCampaignEvents(campaignId, { limit: 20, offset: 0 }));
    });
  }, [campaignId, runAction]);

  const onLoadMargin = useCallback(() => {
    void runAction('margin', async () => {
      if (listMargin && !margin) {
        setMargin(listMargin);
        return;
      }
      setMargin(await getCampaignMargin(campaignId));
    });
  }, [campaignId, listMargin, margin, runAction]);

  const onLoadMappings = useCallback(() => {
    void runAction('mappings', async () => {
      setMappings(await listCampaignConversionMappings(campaignId));
    });
  }, [campaignId, runAction]);

  const onLoadSuggestions = useCallback(() => {
    void runAction('suggestions', async () => {
      const result = await getPlacementBlockSuggestions(campaignId);
      setSuggestions(result.items ?? []);
    });
  }, [campaignId, runAction]);

  const onSaveMappings = useCallback(async () => {
    setSavingMappings(true);
    setActionError(undefined);
    setMappingSaveSuccess(false);
    try {
      const body = { mappings: draftsToMappings(mappingDrafts) };
      const updated = await replaceCampaignConversionMappings(campaignId, body);
      setMappings(updated);
      setMappingSaveSuccess(true);
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingMappings(false);
    }
  }, [campaignId, mappingDrafts]);

  const onBlockPlacement = useCallback(async () => {
    const placement = requireNonEmpty(draftPlacementId, 'Placement ID', 'placement_id');
    if (!placement.ok) {
      setActionError(placement.error);
      return;
    }
    setBlocking(true);
    setActionError(undefined);
    try {
      await blockCampaignPlacement(campaignId, { placement_id: placement.value });
      setDraftPlacementId('');
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setBlocking(false);
    }
  }, [campaignId, draftPlacementId]);

  const onSyncFromPreset = useCallback(async () => {
    const schemaId = campaign?.status_integration_schema_id?.trim();
    if (!schemaId) {
      setActionError(
        validationError('No status integration preset is linked to this campaign.', {
          field: 'status_integration_schema_id',
        })
      );
      return;
    }
    setSyncingPreset(true);
    setActionError(undefined);
    setSyncPresetMessage(undefined);
    try {
      const result = await applyIntegrationSchema(schemaId, { campaign_id: campaignId });
      const count = result.mappings_applied_count;
      setSyncPresetMessage(
        count != null ? `Synced ${count} mapping(s) from preset.` : 'Preset mappings synced.'
      );
      setMappings(await listCampaignConversionMappings(campaignId));
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSyncingPreset(false);
    }
  }, [campaignId, campaign?.status_integration_schema_id]);

  const busy = loadingKey != null || blocking || savingMappings || syncingPreset;

  return {
    draftPlacementId,
    setDraftPlacementId,
    loadingKey,
    stats,
    statsError,
    statsQuery: resolvedStatsQuery,
    events,
    margin,
    mappings,
    mappingDrafts,
    setMappingDrafts,
    suggestions,
    actionError,
    savingMappings,
    mappingSaveSuccess,
    statusIntegrationSchemaName: campaign?.status_integration_schema_name,
    statusIntegrationSchemaId: campaign?.status_integration_schema_id,
    campaign,
    syncingPreset,
    syncPresetMessage,
    blocking,
    busy,
    onLoadStats,
    onLoadEvents,
    onLoadMargin,
    onLoadMappings,
    onLoadSuggestions,
    onSaveMappings,
    onBlockPlacement,
    onSyncFromPreset,
  };
}

export type CampaignOpsPanelWorkspace = ReturnType<typeof useCampaignOpsPanelWorkspace>;
