import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsBlock } from '@/domains/ops/ops_table';
import type { OpsFraudPresetWorkspace } from '@/domains/ops/use_ops_fraud_preset_workspace';
import { useOpsFraudPresetWorkspace } from '@/domains/ops/use_ops_fraud_preset_workspace';
import { adminSpacing } from '@/lib/admin_spacing';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  DirectoryFilterForm,
  FilterField,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { PageSkeleton } from '@/shell/page_skeleton';

export type OpsFraudPresetPanelProps = OpsFraudPresetWorkspace;

export function OpsFraudPresetPanel(workspace: OpsFraudPresetPanelProps) {
  const {
    canList,
    canPatch,
    presets,
    fetching,
    listError,
    hasSnapshot,
    selectedName,
    setSelectedName,
    draftPass,
    setDraftPass,
    draftSuspect,
    setDraftSuspect,
    draftIvt,
    setDraftIvt,
    draftBlock,
    setDraftBlock,
    patching,
    patchError,
    onPatchPreset,
  } = workspace;

  if (!canList) {
    return null;
  }

  if (fetching && !hasSnapshot) {
    return (
      <OpsBlock title="Fraud policy presets">
        <PageSkeleton columns={2} variant="directory" />
      </OpsBlock>
    );
  }

  return (
    <OpsBlock title="Fraud policy presets">
      {listError ? opsPanelError(listError, 'Could not load fraud presets') : null}
      {presets.length === 0 ? (
        <p>No fraud presets returned.</p>
      ) : (
        <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
          <FilterField htmlFor="ops-fraud-preset-name" label="Preset">
            <Select value={selectedName} onValueChange={setSelectedName}>
              <SelectTrigger id="ops-fraud-preset-name">
                <SelectValue placeholder="Select preset" />
              </SelectTrigger>
              <SelectContent>
                {presets.map((preset) => (
                  <SelectItem key={preset.name ?? 'preset'} value={preset.name ?? ''}>
                    {preset.name ?? ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>
          <FilterField htmlFor="ops-fraud-preset-pass" label="Pass threshold">
            <Input
              id="ops-fraud-preset-pass"
              inputMode="numeric"
              value={draftPass}
              onChange={(event) => setDraftPass(event.target.value)}
              disabled={!canPatch}
            />
          </FilterField>
          <FilterField htmlFor="ops-fraud-preset-suspect" label="Suspect threshold">
            <Input
              id="ops-fraud-preset-suspect"
              inputMode="numeric"
              value={draftSuspect}
              onChange={(event) => setDraftSuspect(event.target.value)}
              disabled={!canPatch}
            />
          </FilterField>
          <FilterField htmlFor="ops-fraud-preset-ivt" label="IVT threshold">
            <Input
              id="ops-fraud-preset-ivt"
              inputMode="numeric"
              value={draftIvt}
              onChange={(event) => setDraftIvt(event.target.value)}
              disabled={!canPatch}
            />
          </FilterField>
          <FilterField htmlFor="ops-fraud-preset-block" label="Block threshold">
            <Input
              id="ops-fraud-preset-block"
              inputMode="numeric"
              value={draftBlock}
              onChange={(event) => setDraftBlock(event.target.value)}
              disabled={!canPatch}
            />
          </FilterField>
          <div className={INLINE_FILTER_ACTION_GRID_CLASS}>
            {canPatch ? (
              <Button disabled={patching || !selectedName} onClick={onPatchPreset} type="button">
                {patching ? 'Saving...' : 'Save preset thresholds'}
              </Button>
            ) : (
              <p>Requires shards:write to patch presets.</p>
            )}
          </div>
        </DirectoryFilterForm>
      )}
      {patchError ? opsPanelError(patchError, 'Could not update fraud preset') : null}
    </OpsBlock>
  );
}

export function OpsFraudPresetPanelWithWorkspace() {
  return <OpsFraudPresetPanel {...useOpsFraudPresetWorkspace()} />;
}
