import { DatetimePicker } from '@/components/ui/datetime_picker';
import { FilterField } from '@/shell/filter_panel';

export type FilterDateRangeProps = {
  idFrom: string;
  idTo: string;
  from: string;
  to: string;
  onFromChange: (value: string) => void;
  onToChange: (value: string) => void;
  label?: string;
  /** Pass through to DatetimePicker; default false (day grid only, datetime-local wire). */
  showTime?: boolean;
};

export function FilterDateRange({
  idFrom,
  idTo,
  from,
  to,
  onFromChange,
  onToChange,
  label,
  showTime = false,
}: FilterDateRangeProps) {
  const fromLabel = label ? `${label} from` : 'From';
  const toLabel = label ? `${label} to` : 'To';

  return (
    <>
      <FilterField htmlFor={idFrom} label={fromLabel}>
        <DatetimePicker
          id={idFrom}
          label=""
          showTime={showTime}
          value={from}
          onChange={onFromChange}
        />
      </FilterField>
      <FilterField htmlFor={idTo} label={toLabel}>
        <DatetimePicker id={idTo} label="" showTime={showTime} value={to} onChange={onToChange} />
      </FilterField>
    </>
  );
}
