import { useEffect, useState } from 'react';
import type { CustomerSortField, CustomerSortOrder } from '../../helpers/customers_api.js';
import { Button } from '../system/button.js';
import { Select } from '../system/select.js';
import styles from './customers_directory.module.css';

export type CustomersFilterProps = {
  sort: CustomerSortField;
  order: CustomerSortOrder;
  onApply: (sort: CustomerSortField, order: CustomerSortOrder) => void;
};

const SORT_OPTIONS = [
  { value: 'name', label: 'Name' },
  { value: 'created_at', label: 'Created' },
] as const;

const ORDER_OPTIONS = [
  { value: 'asc', label: 'Ascending' },
  { value: 'desc', label: 'Descending' },
] as const;

export function CustomersFilter({ sort, order, onApply }: CustomersFilterProps) {
  const [draftSort, setDraftSort] = useState<CustomerSortField>(sort);
  const [draftOrder, setDraftOrder] = useState<CustomerSortOrder>(order);

  useEffect(() => {
    setDraftSort(sort);
    setDraftOrder(order);
  }, [sort, order]);

  return (
    <form
      className={styles.filters}
      onSubmit={(event) => {
        event.preventDefault();
        onApply(draftSort, draftOrder);
      }}
    >
      <div className={styles.filterRow}>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Sort</span>
          <Select
            value={draftSort}
            onChange={(value) => setDraftSort(value as CustomerSortField)}
            options={[...SORT_OPTIONS]}
            aria-label="Sort field"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Order</span>
          <Select
            value={draftOrder}
            onChange={(value) => setDraftOrder(value as CustomerSortOrder)}
            options={[...ORDER_OPTIONS]}
            aria-label="Sort order"
          />
        </label>
        <Button type="submit" variant="secondary" size="sm">
          Apply
        </Button>
      </div>
    </form>
  );
}
