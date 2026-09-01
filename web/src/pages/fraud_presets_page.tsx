import { useCallback, useEffect, useState } from 'react';

import { listFraudPresets, patchFraudPreset } from '@/api/fraud_api';
import type { PatchFraudPolicyPresetRequest } from '@/api/types';
import {
  FraudPresets,
  type FraudPresetEditDraft,
} from '@/domains/fraud/fraud_presets';
import { useResource } from '@/hooks/use_resource';

function parseThresholdField(
  label: string,
  raw: string,
): { value: number | undefined; error?: string } {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { value: undefined };
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed < 0 || parsed > 255) {
    return { value: undefined, error: `${label} must be an integer from 0 to 255` };
  }
  return { value: parsed };
}

export function FraudPresetsPage() {
  const [refreshToken, setRefreshToken] = useState(0);
  const [savingPresetName, setSavingPresetName] = useState<string | undefined>();
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [presetDrafts, setPresetDrafts] = useState<Record<string, FraudPresetEditDraft>>({});

  const { data, error, fetching } = useResource(
    (signal) => listFraudPresets(signal),
    [refreshToken],
  );

  useEffect(() => {
    if (!data) {
      return;
    }
    setPresetDrafts((prev) => {
      const next = { ...prev };
      for (const preset of data) {
        const presetName = preset.name ?? '';
        if (!presetName || next[presetName]) {
          continue;
        }
        next[presetName] = {
          pass: preset.pass != null ? String(preset.pass) : '',
          suspect: preset.suspect != null ? String(preset.suspect) : '',
          ivt: preset.ivt != null ? String(preset.ivt) : '',
          block: preset.block != null ? String(preset.block) : '',
        };
      }
      return next;
    });
  }, [data]);

  const onPresetDraftChange = useCallback(
    (name: string, patch: Partial<FraudPresetEditDraft>) => {
      setPresetDrafts((prev) => ({
        ...prev,
        [name]: {
          pass: prev[name]?.pass ?? '',
          suspect: prev[name]?.suspect ?? '',
          ivt: prev[name]?.ivt ?? '',
          block: prev[name]?.block ?? '',
          ...patch,
        },
      }));
    },
    [],
  );

  const onSavePreset = useCallback(
    async (name: string) => {
      const draft = presetDrafts[name];
      if (!name || !draft) {
        return;
      }

      const pass = parseThresholdField('pass', draft.pass);
      if (pass.error) {
        setSaveError(new Error(pass.error));
        setSaveSuccess(false);
        return;
      }
      const suspect = parseThresholdField('suspect', draft.suspect);
      if (suspect.error) {
        setSaveError(new Error(suspect.error));
        setSaveSuccess(false);
        return;
      }
      const ivt = parseThresholdField('ivt', draft.ivt);
      if (ivt.error) {
        setSaveError(new Error(ivt.error));
        setSaveSuccess(false);
        return;
      }
      const block = parseThresholdField('block', draft.block);
      if (block.error) {
        setSaveError(new Error(block.error));
        setSaveSuccess(false);
        return;
      }

      const body: PatchFraudPolicyPresetRequest = {};
      if (pass.value != null) {
        body.pass = pass.value;
      }
      if (suspect.value != null) {
        body.suspect = suspect.value;
      }
      if (ivt.value != null) {
        body.ivt = ivt.value;
      }
      if (block.value != null) {
        body.block = block.value;
      }
      if (
        body.pass == null &&
        body.suspect == null &&
        body.ivt == null &&
        body.block == null
      ) {
        setSaveError(new Error('At least one threshold field is required'));
        setSaveSuccess(false);
        return;
      }

      setSavingPresetName(name);
      setSaveError(undefined);
      setSaveSuccess(false);
      try {
        await patchFraudPreset(name, body);
        setSaveSuccess(true);
        setPresetDrafts((prev) => {
          const next = { ...prev };
          delete next[name];
          return next;
        });
        setRefreshToken((value) => value + 1);
      } catch (err) {
        setSaveError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setSavingPresetName(undefined);
      }
    },
    [presetDrafts],
  );

  return (
    <FraudPresets
      items={data ?? []}
      presetDrafts={presetDrafts}
      fetching={fetching}
      error={error}
      saveError={saveError}
      saveSuccess={saveSuccess}
      hasSnapshot={data != null}
      savingPresetName={savingPresetName}
      onPresetDraftChange={onPresetDraftChange}
      onSavePreset={(name) => void onSavePreset(name)}
    />
  );
}
