import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { deleteRtbDeal, getRtbDeal, patchRtbDeal } from '@/api/rtb_api';
import type { RtbDealUpdateSpec } from '@/api/types';
import { RtbDealEditor } from '@/domains/rtb/rtb_deal_editor';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useBreadcrumbSegmentLabel } from '@/components/system/breadcrumb_context';
import { useResource } from '@/hooks/use_resource';

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

export function RtbDealEditorPage() {
  const navigate = useNavigate();
  const { id } = useParams();
  const dealId = id ? Number(id) : Number.NaN;
  const validId = Number.isFinite(dealId);
  const [reloadKey, setReloadKey] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!validId) {
        return Promise.reject(new Error('Deal ID required'));
      }
      return getRtbDeal(dealId, signal);
    },
    [dealId, validId, reloadKey],
  );

  const [draft, setDraft] = useState<RtbDealUpdateSpec>({});
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>(undefined);
  const [deleteError, setDeleteError] = useState<Error | undefined>(undefined);

  useEffect(() => {
    if (data) {
      setDraft(dealToDraft(data));
    }
  }, [data, reloadKey]);

  const licenseGated = rtbLicenseGated(error) || rtbLicenseGated(saveError) || rtbLicenseGated(deleteError);

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
      setReloadKey((value) => value + 1);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [dealId, draft, validId]);

  const onDelete = useCallback(async () => {
    if (!validId) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await deleteRtbDeal(dealId);
      void navigate('/rtb/deals');
    } catch (err) {
      setDeleteError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeleting(false);
    }
  }, [dealId, navigate, validId]);

  const loadError = useMemo(() => (licenseGated ? undefined : error), [error, licenseGated]);

  useBreadcrumbSegmentLabel(
    validId ? String(dealId) : undefined,
    data?.deal_id ?? (validId ? `Deal ${dealId}` : undefined),
  );

  return (
    <RtbDealEditor
      deal={data}
      draft={draft}
      fetching={fetching}
      saving={saving}
      deleting={deleting}
      error={loadError}
      saveError={saveError}
      deleteError={deleteError}
      hasSnapshot={data != null || licenseGated}
      licenseGated={licenseGated}
      onDraftChange={(patch) => setDraft((prev) => ({ ...prev, ...patch }))}
      onSave={() => {
        void onSave();
      }}
      onDelete={() => {
        void onDelete();
      }}
    />
  );
}
