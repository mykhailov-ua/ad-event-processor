// brand creatives CRUD under /brands/{id}: parallel brand + creatives resources; toast after 2xx mutations.
import { useCallback, useState } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  createBrandCreative,
  deleteBrandCreative,
  getBrand,
  listBrandCreatives,
  patchBrandCreative,
} from '@/api/brands_api';
import type { BrandCreative } from '@/api/types';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { buildBrandCreativeBody } from '@/domains/creative/brand_creative_form';
import type { BrandCreativesNavState } from '@/domains/creative/brand_creatives_nav';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useBrandCreativesPageWorkspace() {
  const { id } = useParams();
  const brandId = id ?? '';
  const location = useLocation();
  const navState = (location.state ?? null) as BrandCreativesNavState | null;
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data: brand } = useResource(
    (signal) => {
      if (!brandId) {
        return Promise.resolve(undefined);
      }
      return getBrand(brandId, signal);
    },
    [brandId]
  );

  const brandName = brand?.name ?? navState?.brandName;
  useBreadcrumbSegmentLabel(brandId || undefined, brandName);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!brandId) {
        return Promise.reject(new Error('Brand ID required'));
      }
      return listBrandCreatives(brandId, signal);
    },
    [brandId, refreshToken]
  );

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [draftWeight, setDraftWeight] = useState('100');
  const [draftStatus, setDraftStatus] = useState('active');
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [actionSuccess, setActionSuccess] = useState(false);

  const listBusy = fetching || acting;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const [editingCreative, setEditingCreative] = useState<BrandCreative | undefined>();
  const [editName, setEditName] = useState('');
  const [editUrl, setEditUrl] = useState('');
  const [editWeight, setEditWeight] = useState('100');
  const [editStatus, setEditStatus] = useState('active');
  const [editSuccess, setEditSuccess] = useState(false);

  const onCreateCreative = useCallback(async () => {
    if (acting) {
      return;
    }
    const built = buildBrandCreativeBody(draftName, draftUrl, draftWeight, draftStatus);
    if (!built.ok) {
      setActionError(new Error(built.error));
      return;
    }
    if (!brandId) {
      setActionError(new Error('Brand ID required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionSuccess(false);
    try {
      await createBrandCreative(brandId, built.body);
      setActionSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Creative created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [acting, brandId, bumpRefreshCoalesced, draftName, draftStatus, draftUrl, draftWeight]);

  const onOpenEditCreative = useCallback((creative: BrandCreative) => {
    setEditingCreative(creative);
    setEditName(creative.name);
    setEditUrl(creative.landing_url);
    setEditWeight(String(creative.weight));
    setEditStatus(creative.status);
    setActionError(undefined);
    setEditSuccess(false);
  }, []);

  const onCloseEditCreative = useCallback(() => {
    setEditingCreative(undefined);
    setEditSuccess(false);
  }, []);

  const onSaveCreative = useCallback(async () => {
    if (acting || !editingCreative) {
      return;
    }
    const built = buildBrandCreativeBody(editName, editUrl, editWeight, editStatus);
    if (!built.ok) {
      setActionError(new Error(built.error));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setEditSuccess(false);
    try {
      await patchBrandCreative(editingCreative.id, built.body);
      setEditSuccess(true);
      toast.success('Creative saved');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [acting, bumpRefreshCoalesced, editName, editStatus, editUrl, editWeight, editingCreative]);

  const onDeleteCreative = useCallback(
    async (creativeId: string) => {
      if (acting) {
        return;
      }
      const row = data?.find((item) => item.id === creativeId);
      const label = row?.name?.trim() || creativeId;
      if (!confirmDestructiveAction(`Delete creative "${label}"?`)) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      try {
        await deleteBrandCreative(creativeId);
        toast.success('Creative deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setActing(false);
      }
    },
    [acting, bumpRefreshCoalesced, data]
  );

  return {
    brandId,
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftName,
    draftUrl,
    draftWeight,
    draftStatus,
    acting,
    actionError,
    actionSuccess,
    editingCreative,
    editName,
    editUrl,
    editWeight,
    editStatus,
    editSuccess,
    onDraftNameChange: setDraftName,
    onDraftUrlChange: setDraftUrl,
    onDraftWeightChange: setDraftWeight,
    onDraftStatusChange: setDraftStatus,
    onEditNameChange: setEditName,
    onEditUrlChange: setEditUrl,
    onEditWeightChange: setEditWeight,
    onEditStatusChange: setEditStatus,
    onCreateCreative,
    onOpenEditCreative,
    onCloseEditCreative,
    onSaveCreative,
    onDeleteCreative,
  };
}
