import {
  AdminSelect,
  type AdminSelectOption,
  type AdminSelectProps,
} from '@/shell/admin_select';

export type CampaignsListFilterOption = AdminSelectOption;
export type CampaignsListFilterSelectProps = AdminSelectProps;

export function CampaignsListFilterSelect(props: CampaignsListFilterSelectProps) {
  return <AdminSelect {...props} />;
}
