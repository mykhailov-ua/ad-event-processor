import { useCallback, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  createBrandCreative,
  deleteBrandCreative,
  listBrandCreatives,
  listBrands,
  patchBrandCreative,
} from '@/api/brands_api';
import type { BrandCreative } from '@/api/types';
import { useBreadcrumbSegmentLabel } from '@/components/system/breadcrumb_context';
import { BrandCreativesDirectory } from '@/domains/creative/brand_creatives_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

export function BrandCreativesPage() {
  const { id } = useParams();
  const brandId = id ?? '';
  const { appliedCustomerId } = useCustomerScope();
  const [reloadToken, setReloadToken] = useState(0);

  const { data: brands } = useResource(
    (signal) => {
      if (!appliedCustomerId) {
        return Promise.resolve([]);
      }
      return listBrands({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId],
  );

  const brandName = useMemo(
    () => brands?.find((brand) => brand.id === brandId)?.name,
    [brandId, brands],
  );
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

  const items = useMemo(() => data ?? [], [data]);

  const onCreateCreative = useCallback(async () => {
    const name = draftName.trim();
    const landingUrl = draftUrl.trim();
    const weight = Number.parseInt(draftWeight.trim(), 10);
    if (!brandId || !name || !landingUrl || !Number.isFinite(weight)) {
      setActionError(new Error('Name, landing URL, and weight are required.'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionSuccess(false);
    try {
      await createBrandCreative(brandId, {
        name,
        landing_url: landingUrl,
        weight,
        status: draftStatus.trim() || 'active',
      });
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
    const name = editName.trim();
    const landingUrl = editUrl.trim();
    const weight = Number.parseInt(editWeight.trim(), 10);
    if (!name || !landingUrl || !Number.isFinite(weight)) {
      setActionError(new Error('Name, landing URL, and weight are required.'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setEditSuccess(false);
    try {
      await patchBrandCreative(editingCreative.id, {
        name,
        landing_url: landingUrl,
        weight,
        status: editStatus.trim() || 'active',
      });
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
      items={items}
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
