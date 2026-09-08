import { useCallback, useEffect, useState } from 'react';

import { getCampaignToggleCohortReport } from '@/api/reports_api';
import type { CampaignToggleCohortKPIRow, CampaignToggleCohortQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export type CampaignToggleField =
  | 'silent_reject_enabled'
  | 'accept_lang_geo_enabled'
  | 'json_serialization_enabled';

function parseToggleField(raw: string | null): CampaignToggleField {
  switch (raw) {
    case 'accept_lang_geo_enabled':
    case 'json_serialization_enabled':
      return raw;
    default:
      return 'silent_reject_enabled';
  }
}

export function useCampaignToggleCohortPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();

  const appliedCampaignId = searchParams.get('campaign_id') ?? '';
  const appliedToggleField = parseToggleField(searchParams.get('toggle_field'));
  const appliedToggleAt = searchParams.get('toggle_at') ?? '';
  const appliedWindowHours = Number.parseInt(searchParams.get('window_hours') ?? '72', 10);

  const [draftCampaignId, setDraftCampaignId] = useState(appliedCampaignId);
  const [draftToggleField, setDraftToggleField] = useState<CampaignToggleField>(appliedToggleField);
  const [draftToggleAt, setDraftToggleAt] = useState(toDatetimeLocalValue(appliedToggleAt));
  const [draftWindowHours, setDraftWindowHours] = useState(
    Number.isFinite(appliedWindowHours) && appliedWindowHours > 0
      ? String(appliedWindowHours)
      : '72'
  );

  useEffect(() => {
    setDraftCampaignId(appliedCampaignId);
    setDraftToggleField(appliedToggleField);
    setDraftToggleAt(toDatetimeLocalValue(appliedToggleAt));
    setDraftWindowHours(
      Number.isFinite(appliedWindowHours) && appliedWindowHours > 0
        ? String(appliedWindowHours)
        : '72'
    );
  }, [appliedCampaignId, appliedToggleAt, appliedToggleField, appliedWindowHours]);

  const shouldFetch = Boolean(appliedCampaignId.trim());

  const { data, error, fetching, revalidating } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      const query: CampaignToggleCohortQuery = {
        campaign_id: appliedCampaignId,
        toggle_field: appliedToggleField,
      };
      if (appliedToggleAt) {
        query.toggle_at = appliedToggleAt;
      }
      if (Number.isFinite(appliedWindowHours) && appliedWindowHours > 0) {
        query.window_hours = appliedWindowHours;
      }
      return getCampaignToggleCohortReport(query, signal);
    },
    [appliedCampaignId, appliedToggleAt, appliedToggleField, appliedWindowHours, shouldFetch]
  );

  const rows: CampaignToggleCohortKPIRow[] = data?.rows ?? [];

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const next = new URLSearchParams();
      const campaignId = draftCampaignId.trim();
      if (campaignId) {
        next.set('campaign_id', campaignId);
      }
      next.set('toggle_field', draftToggleField);
      const toggleAt = fromDatetimeLocalValue(draftToggleAt);
      if (toggleAt) {
        next.set('toggle_at', toggleAt);
      }
      const windowHours = Number.parseInt(draftWindowHours, 10);
      if (Number.isFinite(windowHours) && windowHours > 0) {
        next.set('window_hours', String(windowHours));
      }
      replaceSearchParams(next);
    },
    [draftCampaignId, draftToggleAt, draftToggleField, draftWindowHours, replaceSearchParams]
  );

  return {
    rows,
    insufficient: data?.insufficient_data ?? false,
    toggleAt: data?.toggle_at,
    windowHours: data?.window_hours,
    freshness: data?.freshness,
    draftCampaignId,
    draftToggleField,
    draftToggleAt,
    draftWindowHours,
    fetching: fetching || listQueryPending,
    listRevalidating: revalidating,
    error,
    hasSnapshot: data != null || !shouldFetch,
    onDraftCampaignIdChange: setDraftCampaignId,
    onDraftToggleFieldChange: setDraftToggleField,
    onDraftToggleAtChange: setDraftToggleAt,
    onDraftWindowHoursChange: setDraftWindowHours,
    onApplyFilters,
  };
}
