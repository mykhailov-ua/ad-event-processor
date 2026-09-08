// L3 campaign ops tab: stats/margin/events/mappings/smoke/flow-validate lanes; uses popover stats cache keys.
import { useCallback, useEffect, useMemo, useState } from 'react';

import {
  blockCampaignPlacement,
  getCampaign,
  getCampaignMargin,
  getCampaignStats,
  getPlacementBlockSuggestions,
  listCampaignConversionMappings,
  listCampaignEvents,
  replaceCampaignConversionMappings,
  runCampaignSmoke,
  validateCampaignFlow,
  type CampaignListMetrics,
} from '@/api/campaigns_api';
import { applyIntegrationSchema } from '@/api/integrations_api';
import type {
  CampaignEventListResponse,
  CampaignMargin,
  CampaignStats,
  CampaignStatsQuery,
  ConversionMapping,
  ConversionMappingListResponse,
  PlacementBlockSuggestion,
} from '@/api/types';
import { useResource } from '@/api/use_resource';
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
      const parsed = Number.parseInt(payout, 10);
      if (!Number.isFinite(parsed)) {
        throw new Error('Payout micro must be an integer.');
      }
      mapping.payout_micro = parsed;
    }
    return mapping;
  });
}

type UseCampaignOpsPanelWorkspaceArgs = {
  campaignId: string;
  listMetrics?: CampaignListMetrics;
  listMargin?: CampaignMargin;
  statsQuery?: CampaignStatsQuery;
};

export function useCampaignOpsPanelWorkspace({
  campaignId,
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
  const [smokeMessage, setSmokeMessage] = useState<string | undefined>();
  const [flowMessage, setFlowMessage] = useState<string | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();
  const [savingMappings, setSavingMappings] = useState(false);
  const [mappingSaveSuccess, setMappingSaveSuccess] = useState(false);
  const [syncingPreset, setSyncingPreset] = useState(false);
  const [syncPresetMessage, setSyncPresetMessage] = useState<string | undefined>();

  const { data: campaignMeta } = useResource((signal) => getCampaign(campaignId, signal), [campaignId]);

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
    setSmokeMessage(undefined);
    setFlowMessage(undefined);
    setActionError(undefined);
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

      const fetched = await getCampaignStats(campaignId, resolvedStatsQuery);
      writeCachedCampaignStats(editorCacheKey, fetched);
      setStats(fetched);
    });
  }, [campaignId, listMetrics, resolvedStatsQuery, runAction, stats, statsCacheRevision]);

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

  const onRunSmoke = useCallback(() => {
    void runAction('smoke', async () => {
      const result = await runCampaignSmoke(campaignId);
      setSmokeMessage(
        result.passed ? 'Smoke test passed' : (result.failure_reason ?? 'Smoke test failed')
      );
    });
  }, [campaignId, runAction]);

  const onValidateFlow = useCallback(() => {
    void runAction('flow', async () => {
      const result = await validateCampaignFlow(campaignId);
      setFlowMessage(result.valid ? 'Flow valid' : 'Flow validation failed');
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
    const placementId = draftPlacementId.trim();
    if (!placementId) {
      setActionError(new Error('Placement ID is required.'));
      return;
    }
    setBlocking(true);
    setActionError(undefined);
    try {
      await blockCampaignPlacement(campaignId, { placement_id: placementId });
      setDraftPlacementId('');
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setBlocking(false);
    }
  }, [campaignId, draftPlacementId]);

  const onSyncFromPreset = useCallback(async () => {
    const schemaId = campaignMeta?.status_integration_schema_id?.trim();
    if (!schemaId) {
      setActionError(new Error('No status integration preset is linked to this campaign.'));
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
  }, [campaignId, campaignMeta?.status_integration_schema_id]);

  const busy = loadingKey != null || blocking || savingMappings || syncingPreset;

  return {
    draftPlacementId,
    setDraftPlacementId,
    loadingKey,
    stats,
    events,
    margin,
    mappings,
    mappingDrafts,
    setMappingDrafts,
    suggestions,
    smokeMessage,
    flowMessage,
    actionError,
    savingMappings,
    mappingSaveSuccess,
    statusIntegrationSchemaName: campaignMeta?.status_integration_schema_name,
    statusIntegrationSchemaId: campaignMeta?.status_integration_schema_id,
    syncingPreset,
    syncPresetMessage,
    blocking,
    busy,
    onLoadStats,
    onLoadEvents,
    onLoadMargin,
    onLoadMappings,
    onLoadSuggestions,
    onRunSmoke,
    onValidateFlow,
    onSaveMappings,
    onBlockPlacement,
    onSyncFromPreset,
  };
}

export type CampaignOpsPanelWorkspace = ReturnType<typeof useCampaignOpsPanelWorkspace>;
