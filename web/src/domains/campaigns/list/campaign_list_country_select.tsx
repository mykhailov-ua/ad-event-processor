import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import { SearchableFilterSelect } from '@/shell/searchable_filter_select';

const regionNames =
  typeof Intl !== 'undefined' && 'DisplayNames' in Intl
    ? new Intl.DisplayNames(['en'], { type: 'region' })
    : null;

export function campaignCountryOptionLabel(value: string, fallback: string): string {
  if (value === '__all__') {
    return 'All countries';
  }
  if (!regionNames) {
    return fallback;
  }
  try {
    return regionNames.of(value) ?? fallback;
  } catch {
    return fallback;
  }
}

export type CampaignListCountrySelectProps = {
  'aria-label': string;
  className?: string;
  disabled?: boolean;
  options: CampaignsListFilterOption[];
  title?: string;
  value: string;
  onValueChange?: (value: string) => void;
};

export function CampaignListCountrySelect({
  'aria-label': ariaLabel,
  className,
  disabled = false,
  options,
  title,
  value,
  onValueChange,
}: CampaignListCountrySelectProps) {
  return (
    <SearchableFilterSelect
      aria-label={ariaLabel}
      className={className}
      disabled={disabled}
      formatOptionLabel={campaignCountryOptionLabel}
      options={options}
      searchAriaLabel="Search countries"
      searchPlaceholder="All countries"
      title={title}
      value={value}
      onValueChange={onValueChange}
    />
  );
}
