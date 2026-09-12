import { useMemo } from 'react';

import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select';
import { EXPORT_HUB_SEARCH_THRESHOLD } from '@/domains/exports/export_hub_limits';
import { FilterField } from '@/shell/filter_panel';
import { type ExportHubEntry, exportHubCatalogOptions } from '@/domains/exports/export_hub_catalog';
import { SearchableFilterSelect } from '@/shell/searchable_filter_select';

export type ExportHubCatalogPickerOption = {
  value: string;
  label: string;
};

export type ExportHubCatalogPickerProps = {
  entries: ExportHubEntry[];
  value: string;
  disabled?: boolean;
  onValueChange: (value: string) => void;
};

export function exportHubCatalogPickerOptions(
  entries: ExportHubEntry[]
): ExportHubCatalogPickerOption[] {
  return exportHubCatalogOptions(entries);
}

export function ExportHubCatalogPicker({
  entries,
  value,
  disabled = false,
  onValueChange,
}: ExportHubCatalogPickerProps) {
  const options = useMemo(() => exportHubCatalogOptions(entries), [entries]);
  const selectedLabel = options.find((option) => option.value === value)?.label;
  const showSearch = options.length > EXPORT_HUB_SEARCH_THRESHOLD;

  if (showSearch) {
    return (
      <FilterField htmlFor="export-hub-catalog" label="Export">
        <SearchableFilterSelect
          aria-label="Choose export target"
          disabled={disabled}
          matchPopoverToTrigger
          options={options}
          searchPlaceholder="Search exports"
          showSearch
          triggerId="export-hub-catalog"
          value={value}
          onValueChange={onValueChange}
        />
      </FilterField>
    );
  }

  return (
    <FilterField htmlFor="export-hub-catalog" label="Export">
      <Select disabled={disabled} value={value || undefined} onValueChange={onValueChange}>
        <SelectTrigger id="export-hub-catalog">
          <span>{selectedLabel ?? 'Select export'}</span>
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FilterField>
  );
}
