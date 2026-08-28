import type { Customer, CustomerSortField, CustomerSortOrder } from '../../helpers/customers_api.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { CustomersFilter } from './customers_filter.js';
import { CustomersGrid } from './customers_grid.js';
import { CustomersToolbar } from './customers_toolbar.js';
import styles from './customers_directory.module.css';

export type CustomersDirectoryProps = {
  items: Customer[];
  total: number;
  limit: number;
  offset: number;
  sort: CustomerSortField;
  order: CustomerSortOrder;
  loading: boolean;
  error: unknown;
  onSortHeader: (field: CustomerSortField) => void;
  onFilterApply: (sort: CustomerSortField, order: CustomerSortOrder) => void;
  onOffsetChange: (offset: number) => void;
};

export function CustomersDirectory({
  items,
  total,
  limit,
  offset,
  sort,
  order,
  loading,
  error,
  onSortHeader,
  onFilterApply,
  onOffsetChange,
}: CustomersDirectoryProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load customers" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome
        title="Customers"
        badge={<LoadingCountBadge loading={loading} label={`${total} total`} />}
      />
      <CustomersToolbar />
      <CustomersFilter sort={sort} order={order} onApply={onFilterApply} />
      <div className={styles.content}>
        <CustomersGrid
          items={items}
          loading={loading}
          sort={sort}
          order={order}
          onSortHeader={onSortHeader}
        />
      </div>
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={total} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
