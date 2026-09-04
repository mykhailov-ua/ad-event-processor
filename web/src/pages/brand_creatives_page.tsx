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
import { BrandCreativesDirectory } from '@/domains/creative/brand_creatives_directory';
import { buildBrandCreativeBody } from '@/domains/creative/brand_creative_form';
import type { BrandCreativesNavState } from '@/domains/creative/brand_creatives_nav';
import { useResource } from '@/api/use_resource';

export function BrandCreativesPage() {
  const { id } = useParams();
  const brandId = id ?? '';
  const location = useLocation();
  const navState = (location.state ?? null) as BrandCreativesNavState | null;
  const [reloadToken, setReloadToken] = useState(0);

  const { data: brand } = useResource(
    (signal) => {
      if (!brandId) {
        return Promise.resolve(undefined);
      }
      return getBrand(brandId, signal);
    },
    [brandId],
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
    [brandId, reloadToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [draftWeight, setDraftWeight] = useState('100');
  const [draftStatus, setDraftStatus] = useState('active');
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [actionSuccess, setActionSuccess] = useState(false);

  const [editingCreative, setEditingCreative] = useState<BrandCreative | undefined>();
  const [editName, setEditName] = useState('');
  const [editUrl, setEditUrl] = useState('');
  const [editWeight, setEditWeight] = useState('100');
  const [editStatus, setEditStatus] = useState('active');
  const [editSuccess, setEditSuccess] = useState(false);

  const onCreateCreative = useCallback(async () => {
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
      setReloadToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [brandId, draftName, draftStatus, draftUrl, draftWeight]);

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
    if (!editingCreative) {
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
      setReloadToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [editName, editStatus, editUrl, editWeight, editingCreative]);

  const onDeleteCreative = useCallback(
    async (creativeId: string) => {
      setActing(true);
      setActionError(undefined);
      try {
        await deleteBrandCreative(creativeId);
        toast.success('Creative deleted');
        setReloadToken((value) => value + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setActing(false);
      }
    },
    [],
  );

  return (
    <BrandCreativesDirectory
      brandId={brandId}
      items={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftName={draftName}
      draftUrl={draftUrl}
      draftWeight={draftWeight}
      draftStatus={draftStatus}
      acting={acting}
      actionError={actionError}
      actionSuccess={actionSuccess}
      editingCreative={editingCreative}
      editName={editName}
      editUrl={editUrl}
      editWeight={editWeight}
      editStatus={editStatus}
      editSuccess={editSuccess}
      onDraftNameChange={setDraftName}
      onDraftUrlChange={setDraftUrl}
      onDraftWeightChange={setDraftWeight}
      onDraftStatusChange={setDraftStatus}
      onEditNameChange={setEditName}
      onEditUrlChange={setEditUrl}
      onEditWeightChange={setEditWeight}
      onEditStatusChange={setEditStatus}
      onCreateCreative={() => {
        void onCreateCreative();
      }}
      onOpenEditCreative={onOpenEditCreative}
      onCloseEditCreative={onCloseEditCreative}
      onSaveCreative={() => {
        void onSaveCreative();
      }}
      onDeleteCreative={(creativeId) => {
        void onDeleteCreative(creativeId);
      }}
    />
  );
}
