import { useCallback, useMemo } from 'react';
import type { Campaign, CampaignSortField } from '../../helpers/campaigns_api.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { CampaignsBulkBar } from './campaigns_bulk_bar.js';
import { CampaignsFilter, type CampaignsFilterValues } from './campaigns_filter.js';
import { CampaignsGrid } from './campaigns_grid.js';
import { CampaignsToolbar } from './campaigns_toolbar.js';
import styles from './campaigns_directory.module.css';

export type CampaignsDirectoryProps = {
  items: Campaign[];
  total: number;
  limit: number;
  offset: number;
  filterValues: CampaignsFilterValues;
  loading: boolean;
  error: unknown;
  canBulk: boolean;
  customerScoped: boolean;
  selectedIds: Set<string>;
  onFilterApply: (values: CampaignsFilterValues) => void;
  onSortHeader: (field: CampaignSortField) => void;
  onOffsetChange: (offset: number) => void;
  onToggleRow: (id: string, checked: boolean) => void;
  onToggleAll: (checked: boolean, ids: string[]) => void;
  onClearSelection: () => void;
  onBulkSuccess: () => void;
};

export function CampaignsDirectory({
  items,
  total,
  limit,
  offset,
  filterValues,
  loading,
  error,
  canBulk,
  customerScoped,
  selectedIds,
  onFilterApply,
  onSortHeader,
  onOffsetChange,
  onToggleRow,
  onToggleAll,
  onClearSelection,
  onBulkSuccess,
}: CampaignsDirectoryProps) {
  const selectedList = useMemo(() => Array.from(selectedIds), [selectedIds]);

  const handleSortHeader = useCallback(
    (field: CampaignSortField) => {
      onSortHeader(field);
    },
    [onSortHeader]
  );

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load campaigns" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome
        title="Campaigns"
        badge={<LoadingCountBadge loading={loading} label={`${total} total`} />}
      />
      <CampaignsToolbar />
      <CampaignsFilter values={filterValues} onApply={onFilterApply} />
      {canBulk ? (
        <CampaignsBulkBar
          selectedIds={selectedList}
          onClear={onClearSelection}
          onSuccess={onBulkSuccess}
        />
      ) : null}
      <div className={styles.content}>
        <CampaignsGrid
          items={items}
          loading={loading}
          sort={filterValues.sort}
          order={filterValues.order}
          selectedIds={selectedIds}
          canBulk={canBulk}
          customerScoped={customerScoped}
          onSortHeader={handleSortHeader}
          onToggleRow={onToggleRow}
          onToggleAll={onToggleAll}
        />
      </div>
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={total} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
