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
  FLOW_PATH_ROTATION_OPTIONS,
  newFlowEntityRef,
  newFlowPathRow,
  normalizeEntityRefWeights,
  normalizeVisualPathWeights,
  type FlowEntityRefRow,
  type FlowPathRotationMode,
  type FlowPathVisualRow,
} from '@/domains/creative/flow_path_model';
import { adminTypography } from '@/lib/admin_spacing';

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
        <SecondaryActionButton
          disabled={disabled}
          onClick={onAddRow}
          type="button"
          variant="secondary"
        >
          Add path
        </SecondaryActionButton>
      </div>

      {validationError ? <ErrorBlock message={validationError} title="Flow validation" /> : null}

      {rows.map((row, index) => (
        <section key={row.row_id} className="grid gap-3 rounded-md border border-border p-3">
          <div className="flex items-center justify-between gap-2">
            <h3 className={adminTypography.sectionTitle}>Path {index + 1}</h3>
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

          <FilterField htmlFor={`flow-weight-${row.row_id}`} label="Path weight %">
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

          <FilterField htmlFor={`flow-rotation-${row.row_id}`} label="Rotation mode">
            <Select
              disabled={disabled}
              value={row.rotation_mode}
              onValueChange={(value) =>
                updateRow(index, { rotation_mode: value as FlowPathRotationMode })
              }
            >
              <SelectTrigger id={`flow-rotation-${row.row_id}`}>
                <SelectValue placeholder="Weighted" />
              </SelectTrigger>
              <SelectContent>
                {FLOW_PATH_ROTATION_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FlowEntityRefEditor
            disabled={disabled}
            entities={landers}
            entityLabel="Lander"
            idPrefix={`flow-lander-${row.row_id}`}
            refs={row.landers}
            onRefsChange={(landers) => updateRow(index, { landers })}
          />

          <FlowEntityRefEditor
            disabled={disabled}
            entities={offers}
            entityLabel="Offer"
            idPrefix={`flow-offer-${row.row_id}`}
            refs={row.offers}
            onRefsChange={(offers) => updateRow(index, { offers })}
          />

          <FilterField htmlFor={`flow-countries-${row.row_id}`} label="Geo (countries)">
            <Input
              disabled={disabled}
              id={`flow-countries-${row.row_id}`}
              placeholder="US, CA"
              value={row.countries}
              onChange={(event) => updateRow(index, { countries: event.target.value })}
            />
          </FilterField>

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
                        const devices =
                          next === true
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

type FlowEntityRefEditorProps = {
  refs: FlowEntityRefRow[];
  entities: Array<{ id: string; name: string }>;
  entityLabel: string;
  idPrefix: string;
  disabled?: boolean;
  onRefsChange: (refs: FlowEntityRefRow[]) => void;
};

function FlowEntityRefEditor({
  refs,
  entities,
  entityLabel,
  idPrefix,
  disabled = false,
  onRefsChange,
}: FlowEntityRefEditorProps) {
  const updateRef = (refIndex: number, patch: Partial<FlowEntityRefRow>) => {
    onRefsChange(refs.map((ref, index) => (index === refIndex ? { ...ref, ...patch } : ref)));
  };

  const onAddRef = () => {
    const even = Math.floor(100 / (refs.length + 1));
    const next = [...refs, newFlowEntityRef(even)].map((ref) => ({ ...ref, weight: even }));
    onRefsChange(normalizeEntityRefWeights(next));
  };

  const onRemoveRef = (refIndex: number) => {
    if (refs.length <= 1) {
      return;
    }
    onRefsChange(normalizeEntityRefWeights(refs.filter((_, index) => index !== refIndex)));
  };

  return (
    <div className="grid gap-2">
      <div className="flex items-center justify-between gap-2">
        <Label>{entityLabel}s</Label>
        <SecondaryActionButton
          disabled={disabled}
          onClick={onAddRef}
          type="button"
          variant="secondary"
        >
          Add {entityLabel.toLowerCase()}
        </SecondaryActionButton>
      </div>
      {refs.map((ref, refIndex) => (
        <div
          key={ref.ref_id}
          className="grid gap-2 rounded-md border border-border/60 p-2 sm:grid-cols-[1fr_6rem_auto]"
        >
          <FilterField
            htmlFor={`${idPrefix}-${ref.ref_id}`}
            label={`${entityLabel} ${refIndex + 1}`}
          >
            <Select
              disabled={disabled}
              value={ref.entity_id || undefined}
              onValueChange={(value) => updateRef(refIndex, { entity_id: value })}
            >
              <SelectTrigger id={`${idPrefix}-${ref.ref_id}`}>
                <SelectValue placeholder={`Select ${entityLabel.toLowerCase()}`} />
              </SelectTrigger>
              <SelectContent>
                {entities.map((entity) => (
                  <SelectItem key={entity.id} value={entity.id}>
                    {entity.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>
          <FilterField htmlFor={`${idPrefix}-weight-${ref.ref_id}`} label="Weight %">
            <Input
              disabled={disabled}
              id={`${idPrefix}-weight-${ref.ref_id}`}
              inputMode="decimal"
              value={String(ref.weight)}
              onChange={(event) =>
                updateRef(refIndex, { weight: Number.parseFloat(event.target.value) || 0 })
              }
            />
          </FilterField>
          {refs.length > 1 ? (
            <SecondaryActionButton
              disabled={disabled}
              onClick={() => onRemoveRef(refIndex)}
              type="button"
              variant="secondary"
            >
              Remove
            </SecondaryActionButton>
          ) : null}
        </div>
      ))}
    </div>
  );
}
