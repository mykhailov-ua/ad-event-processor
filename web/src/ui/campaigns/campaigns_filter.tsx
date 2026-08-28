import { useEffect, useState } from 'react';
import type { CampaignSortField, CampaignSortOrder } from '../../helpers/campaigns_api.js';
import { Button } from '../system/button.js';
import { FieldInput } from '../system/field_input.js';
import { Select } from '../system/select.js';
import styles from './campaigns_directory.module.css';

export type CampaignsFilterValues = {
  customer_id: string;
  status: string;
  q: string;
  pacing_mode: string;
  sort: CampaignSortField;
  order: CampaignSortOrder;
};

export type CampaignsFilterProps = {
  values: CampaignsFilterValues;
  onApply: (values: CampaignsFilterValues) => void;
};

const SORT_OPTIONS = [
  { value: 'name', label: 'Name' },
  { value: 'updated_at', label: 'Updated' },
  { value: 'spend', label: 'Spend' },
] as const;

const ORDER_OPTIONS = [
  { value: 'asc', label: 'Ascending' },
  { value: 'desc', label: 'Descending' },
] as const;

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'active', label: 'Active' },
  { value: 'paused', label: 'Paused' },
  { value: 'draft', label: 'Draft' },
  { value: 'archived', label: 'Archived' },
] as const;

const PACING_OPTIONS = [
  { value: '', label: 'All pacing' },
  { value: 'even', label: 'Even' },
  { value: 'asap', label: 'ASAP' },
  { value: 'front_loaded', label: 'Front-load' },
] as const;

export function CampaignsFilter({ values, onApply }: CampaignsFilterProps) {
  const [draft, setDraft] = useState<CampaignsFilterValues>(values);

  useEffect(() => {
    setDraft(values);
  }, [values]);

  return (
    <form
      className={styles.filters}
      onSubmit={(event) => {
        event.preventDefault();
        onApply(draft);
      }}
    >
      <div className={styles.filterRow}>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Customer ID</span>
          <FieldInput
            type="text"
            value={draft.customer_id}
            onChange={(event) => setDraft((prev) => ({ ...prev, customer_id: event.target.value }))}
            placeholder="UUID"
            aria-label="Customer ID filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Status</span>
          <Select
            value={draft.status}
            onChange={(value) => setDraft((prev) => ({ ...prev, status: value }))}
            options={[...STATUS_OPTIONS]}
            aria-label="Status filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Search</span>
          <FieldInput
            type="search"
            value={draft.q}
            onChange={(event) => setDraft((prev) => ({ ...prev, q: event.target.value }))}
            placeholder="Name"
            aria-label="Search campaigns"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Pacing</span>
          <Select
            value={draft.pacing_mode}
            onChange={(value) => setDraft((prev) => ({ ...prev, pacing_mode: value }))}
            options={[...PACING_OPTIONS]}
            aria-label="Pacing mode filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Sort</span>
          <Select
            value={draft.sort}
            onChange={(value) => setDraft((prev) => ({ ...prev, sort: value as CampaignSortField }))}
            options={[...SORT_OPTIONS]}
            aria-label="Sort field"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Order</span>
          <Select
            value={draft.order}
            onChange={(value) => setDraft((prev) => ({ ...prev, order: value as CampaignSortOrder }))}
            options={[...ORDER_OPTIONS]}
            aria-label="Sort order"
          />
        </label>
        <Button type="submit" variant="secondary">
          Apply
        </Button>
      </div>
    </form>
  );
}
