// RTB deal editor: GET/PATCH/DELETE single deal; local draft until save.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { deleteRtbDeal, getRtbDeal, patchRtbDeal } from '@/api/rtb_api';
import type { RtbDealUpdateSpec } from '@/api/types';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

function dealToDraft(deal: {
  deal_id?: string;
  floor_micro?: number;
  geo_mask?: number;
  cat_mask?: number;
  pacing?: string;
  seats?: number;
  customer_id?: string;
}): RtbDealUpdateSpec {
  return {
    deal_id: deal.deal_id,
    floor_micro: deal.floor_micro,
    geo_mask: deal.geo_mask,
    cat_mask: deal.cat_mask,
    pacing: deal.pacing,
    seats: deal.seats,
    customer_id: deal.customer_id,
  };
}

export function useRtbDealEditorPageWorkspace() {
  const navigate = useNavigate();
  const { id } = useParams();
  const dealId = id ? Number(id) : Number.NaN;
  const validId = Number.isFinite(dealId);
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!validId) {
        return Promise.reject(new Error('Deal ID required'));
      }
      return getRtbDeal(dealId, signal);
    },
    [dealId, refreshToken, validId]
  );

  const [draft, setDraft] = useState<RtbDealUpdateSpec>({});
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>(undefined);
  const [deleteError, setDeleteError] = useState<Error | undefined>(undefined);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || saving || deleting);

  useEffect(() => {
    if (data) {
      setDraft(dealToDraft(data));
    }
  }, [data]);

  const licenseGated =
    rtbLicenseGated(error) || rtbLicenseGated(saveError) || rtbLicenseGated(deleteError);

  const onSave = useCallback(async () => {
    if (!validId) {
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    try {
      const body: RtbDealUpdateSpec = {
        deal_id: draft.deal_id,
        floor_micro: draft.floor_micro,
        geo_mask: draft.geo_mask,
        cat_mask: draft.cat_mask,
        pacing: draft.pacing,
        seats: draft.seats,
        customer_id: draft.customer_id,
      };
      await patchRtbDeal(dealId, body);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [bumpRefreshCoalesced, dealId, draft, validId]);

  const onDelete = useCallback(async () => {
    if (!validId) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await deleteRtbDeal(dealId);
      void navigate('/rtb/deals');
    } catch (err: unknown) {
      setDeleteError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeleting(false);
    }
  }, [dealId, navigate, validId]);

  const loadError = useMemo(() => (licenseGated ? undefined : error), [error, licenseGated]);

  useBreadcrumbSegmentLabel(
    validId ? String(dealId) : undefined,
    data?.deal_id ?? (validId ? `Deal ${dealId}` : undefined)
  );

  return {
    deal: data,
    draft,
    fetching,
    saving,
    deleting,
    error: loadError,
    saveError,
    deleteError,
    hasSnapshot: data != null || licenseGated,
    licenseGated,
    onDraftChange: (patch: Partial<RtbDealUpdateSpec>) =>
      setDraft((prev) => ({ ...prev, ...patch })),
    onSave: () => {
      void onSave();
    },
    onDelete: () => {
      void onDelete();
    },
  };
}
