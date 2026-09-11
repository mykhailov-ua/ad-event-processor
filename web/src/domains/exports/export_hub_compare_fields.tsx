import { DatetimePicker } from '@/components/ui/datetime_picker';
import { adminTypography } from '@/lib/admin_kit';
import { FilterField } from '@/shell/filter_panel';

export type ExportHubCompareFieldsProps = {
  disabled?: boolean;
  compareFrom: string;
  compareTo: string;
  onCompareFromChange: (value: string) => void;
  onCompareToChange: (value: string) => void;
};

export function ExportHubCompareFields({
  disabled,
  compareFrom,
  compareTo,
  onCompareFromChange,
  onCompareToChange,
}: ExportHubCompareFieldsProps) {
  return (
    <>
      <DatetimePicker
        disabled={disabled}
        id="export-hub-compare-from"
        label="Compare from"
        value={compareFrom}
        onChange={onCompareFromChange}
      />
      <DatetimePicker
        disabled={disabled}
        id="export-hub-compare-to"
        label="Compare to"
        value={compareTo}
        onChange={onCompareToChange}
      />
      <p className={adminTypography.bodyMuted}>
        Optional second date range. Delta columns appear for supported reports (for example
        placements).
      </p>
    </>
  );
}
