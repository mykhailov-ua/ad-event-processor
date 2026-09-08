import { useMemo } from 'react';

import { SearchableFilterSelect } from '@/shell/searchable_filter_select';

export type CustomerComboboxOption = {
  id: string;
  name: string;
};

export type CustomerComboboxProps = {
  id?: string;
  value: string;
  options: CustomerComboboxOption[];
  loading?: boolean;
  disabled?: boolean;
  onValueChange: (customerId: string) => void;
};

export function CustomerCombobox({
  id,
  value,
  options,
  loading = false,
  disabled = false,
  onValueChange,
}: CustomerComboboxProps) {
  const selectOptions = useMemo(
    () => [
      { value: '', label: 'All customers' },
      ...options.map((customer) => ({
        value: customer.id,
        label: customer.name,
      })),
    ],
    [options]
  );

  const resolvedValue = value || '';
  const selected =
    selectOptions.find((option) => option.value === resolvedValue) ??
    (resolvedValue
      ? { value: resolvedValue, label: resolvedValue }
      : selectOptions[0]);

  const displayOptions = useMemo(() => {
    if (!resolvedValue || selectOptions.some((option) => option.value === resolvedValue)) {
      return selectOptions;
    }
    return [...selectOptions, selected];
  }, [resolvedValue, selectOptions, selected]);

  return (
    <SearchableFilterSelect
      aria-label="Customer"
      disabled={disabled || loading}
      options={displayOptions}
      searchPlaceholder={loading ? 'Loading customers...' : 'All customers'}
      triggerId={id}
      value={resolvedValue}
      onValueChange={onValueChange}
    />
  );
}
