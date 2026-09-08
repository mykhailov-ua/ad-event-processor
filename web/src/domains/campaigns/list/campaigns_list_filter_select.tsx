import { AdminSelect, type AdminSelectOption, type AdminSelectProps } from '@/shell/admin_select';
import {
  SearchableFilterSelect,
  type SearchableFilterOption,
} from '@/shell/searchable_filter_select';

export type CampaignsListFilterOption = AdminSelectOption;
export type CampaignsListFilterSelectProps = AdminSelectProps;

export function CampaignsListFilterSelect(props: CampaignsListFilterSelectProps) {
  return <AdminSelect {...props} />;
}

export type CampaignsListSearchableFilterSelectProps = CampaignsListFilterSelectProps & {
  searchPlaceholder?: string;
  searchAriaLabel?: string;
  formatOptionLabel?: (value: string, fallback: string) => string;
};

export function CampaignsListSearchableFilterSelect({
  searchPlaceholder,
  searchAriaLabel,
  formatOptionLabel,
  ...props
}: CampaignsListSearchableFilterSelectProps) {
  const fallbackPlaceholder =
    props.options.find((option) => option.value === props.value)?.label ??
    props.options[0]?.label ??
    'Search...';

  return (
    <SearchableFilterSelect
      {...props}
      formatOptionLabel={formatOptionLabel}
      options={props.options as SearchableFilterOption[]}
      searchAriaLabel={searchAriaLabel ?? props['aria-label']}
      searchPlaceholder={searchPlaceholder ?? fallbackPlaceholder}
    />
  );
}
