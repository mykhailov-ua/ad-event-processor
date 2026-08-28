import * as auth from '../../helpers/auth.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import type { Brand } from '../../helpers/brands_api.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { Button } from '../system/button.js';
import { BrandCreateModal } from './brand_create_modal.js';
import { BrandsFilter } from './brands_filter.js';
import { BrandsGrid } from './brands_grid.js';
import styles from './brands_directory.module.css';

export type BrandsDirectoryProps = {
  customerId: string;
  items: Brand[];
  loading: boolean;
  error: unknown;
  expandedBrandId: string | null;
  modalOpen: boolean;
  modalBusy: boolean;
  modalError: string | null;
  onCustomerApply: (customerId: string) => void;
  onToggleExpand: (brandId: string) => void;
  onOpenCreate: () => void;
  onCloseModal: () => void;
  onSubmitCreate: (body: { customer_id: string; name: string }) => void;
  onReload: () => void;
};

export function BrandsDirectory({
  customerId,
  items,
  loading,
  error,
  expandedBrandId,
  modalOpen,
  modalBusy,
  modalError,
  onCustomerApply,
  onToggleExpand,
  onOpenCreate,
  onCloseModal,
  onSubmitCreate,
  onReload,
}: BrandsDirectoryProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Brands access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load brands" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome title="Brands" badge={<LoadingCountBadge loading={loading} label={`${items.length} brands`} />} />
      <BrandsFilter customerId={customerId} onApply={onCustomerApply} />
      {canWrite && customerId ? (
        <div className={styles.toolbar}>
          <Button variant="primary" onClick={onOpenCreate}>
            Create brand
          </Button>
        </div>
      ) : null}
      <div className={styles.content}>
        {!customerId ? (
          <div className={styles.hint}>Enter a customer ID and apply to load brands.</div>
        ) : (
          <BrandsGrid
            items={items}
            loading={loading}
            expandedBrandId={expandedBrandId}
            canWrite={canWrite}
            onToggleExpand={onToggleExpand}
            onReload={onReload}
          />
        )}
      </div>
      <BrandCreateModal
        open={modalOpen}
        customerId={customerId}
        busy={modalBusy}
        error={modalError}
        onClose={onCloseModal}
        onSubmit={onSubmitCreate}
      />
    </div>
  );
}
