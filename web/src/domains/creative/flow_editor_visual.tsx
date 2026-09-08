import { SecondaryActionButton } from '@/shell/action_buttons';
import { FilterField } from '@/shell/filter_panel';
import { ErrorBlock } from '@/shell/error_block';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { Lander, Offer } from '@/api/types';
import {
  applySplitPreset,
  newFlowPathRow,
  normalizeVisualPathWeights,
  type FlowPathVisualRow,
} from '@/domains/creative/flow_path_model';

const DEVICE_OPTIONS = ['desktop', 'mobile', 'tablet'] as const;

export type FlowEditorVisualProps = {
  rows: FlowPathVisualRow[];
  landers: Lander[];
  offers: Offer[];
  validationError?: string;
  disabled?: boolean;
  onRowsChange: (rows: FlowPathVisualRow[]) => void;
};

export function FlowEditorVisual({
  rows,
  landers,
  offers,
  validationError,
  disabled = false,
  onRowsChange,
}: FlowEditorVisualProps) {
  const updateRow = (index: number, patch: Partial<FlowPathVisualRow>) => {
    onRowsChange(rows.map((row, rowIndex) => (rowIndex === index ? { ...row, ...patch } : row)));
  };

  const onAddRow = () => {
    const even = Math.floor(100 / (rows.length + 1));
    const next = [...rows, newFlowPathRow()].map((row) => ({ ...row, weight: even }));
    onRowsChange(normalizeVisualPathWeights(next));
  };

  const onRemoveRow = (index: number) => {
    if (rows.length <= 1) {
      return;
    }
    onRowsChange(normalizeVisualPathWeights(rows.filter((_, rowIndex) => rowIndex !== index)));
  };

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap gap-2">
        <SecondaryActionButton
          disabled={disabled}
          onClick={() => onRowsChange(applySplitPreset(rows, [50, 50]))}
          type="button"
          variant="secondary"
        >
          50 / 50 split
        </SecondaryActionButton>
        <SecondaryActionButton
          disabled={disabled}
          onClick={() => onRowsChange(applySplitPreset(rows, [70, 30]))}
          type="button"
          variant="secondary"
        >
          70 / 30 split
        </SecondaryActionButton>
        <SecondaryActionButton
          disabled={disabled}
          onClick={() => onRowsChange(normalizeVisualPathWeights(rows))}
          type="button"
          variant="secondary"
        >
          Normalize weights
        </SecondaryActionButton>
        <SecondaryActionButton disabled={disabled} onClick={onAddRow} type="button" variant="secondary">
          Add path
        </SecondaryActionButton>
      </div>

      {validationError ? <ErrorBlock message={validationError} title="Flow validation" /> : null}

      {rows.map((row, index) => (
        <section
          key={row.row_id}
          className="grid gap-3 rounded-md border border-border p-3"
        >
          <div className="flex items-center justify-between gap-2">
            <h3 className="text-sm font-semibold">Path {index + 1}</h3>
            {rows.length > 1 ? (
              <SecondaryActionButton
                disabled={disabled}
                onClick={() => onRemoveRow(index)}
                type="button"
                variant="secondary"
              >
                Remove
              </SecondaryActionButton>
            ) : null}
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <FilterField htmlFor={`flow-weight-${row.row_id}`} label="Weight %">
              <Input
                disabled={disabled}
                id={`flow-weight-${row.row_id}`}
                inputMode="decimal"
                value={String(row.weight)}
                onChange={(event) =>
                  updateRow(index, { weight: Number.parseFloat(event.target.value) || 0 })
                }
              />
            </FilterField>

            <FilterField htmlFor={`flow-lander-${row.row_id}`} label="Lander">
              <Select
                disabled={disabled}
                value={row.lander_id || undefined}
                onValueChange={(value) => updateRow(index, { lander_id: value })}
              >
                <SelectTrigger id={`flow-lander-${row.row_id}`}>
                  <SelectValue placeholder="Select lander" />
                </SelectTrigger>
                <SelectContent>
                  {landers.map((lander) => (
                    <SelectItem key={lander.id} value={lander.id}>
                      {lander.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>

            <FilterField htmlFor={`flow-offer-${row.row_id}`} label="Offer">
              <Select
                disabled={disabled}
                value={row.offer_id || undefined}
                onValueChange={(value) => updateRow(index, { offer_id: value })}
              >
                <SelectTrigger id={`flow-offer-${row.row_id}`}>
                  <SelectValue placeholder="Select offer" />
                </SelectTrigger>
                <SelectContent>
                  {offers.map((offer) => (
                    <SelectItem key={offer.id} value={offer.id}>
                      {offer.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>

            <FilterField htmlFor={`flow-countries-${row.row_id}`} label="Geo (countries)">
              <Input
                disabled={disabled}
                id={`flow-countries-${row.row_id}`}
                placeholder="US, CA"
                value={row.countries}
                onChange={(event) => updateRow(index, { countries: event.target.value })}
              />
            </FilterField>
          </div>

          <div className="grid gap-2">
            <Label>Device filters</Label>
            <div className="flex flex-wrap gap-3">
              {DEVICE_OPTIONS.map((device) => {
                const checked = row.devices.includes(device);
                return (
                  <div key={device} className="flex items-center gap-2">
                    <Checkbox
                      checked={checked}
                      disabled={disabled}
                      id={`flow-device-${row.row_id}-${device}`}
                      onCheckedChange={(next) => {
                        const devices = next === true
                          ? [...row.devices, device]
                          : row.devices.filter((value) => value !== device);
                        updateRow(index, { devices });
                      }}
                    />
                    <Label className="font-normal" htmlFor={`flow-device-${row.row_id}-${device}`}>
                      {device}
                    </Label>
                  </div>
                );
              })}
            </div>
          </div>
        </section>
      ))}
    </div>
  );
}
