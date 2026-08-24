import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  FRAUD_PRESET_OPTIONS,
  fetchCampaignFraudConfig,
  fetchFraudPresets,
  patchCampaignFraudConfig,
  previewCampaignFraudImpact,
  type CampaignFraudConfig,
  type CampaignFraudPreview,
  type FraudPolicyPreset,
  type FraudSensitivityPreset,
} from '../helpers/fraud_api.js';
import { fraudTierBandRowsFromThresholds } from '../helpers/edge_fraud_tier.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';
import { Checkbox } from './checkbox.js';

export type CampaignFraudSectionProps = {
  campaignId: string;
  canWrite: boolean;
  onCampaignFlagsChanged?: () => void;
};

type ThresholdField =
  | 'fraud_threshold_pass'
  | 'fraud_threshold_suspect'
  | 'fraud_threshold_ivt'
  | 'fraud_threshold_block';

const THRESHOLD_FIELDS: Array<{ key: ThresholdField; label: string }> = [
  { key: 'fraud_threshold_pass', label: 'Pass max' },
  { key: 'fraud_threshold_suspect', label: 'Suspect max' },
  { key: 'fraud_threshold_ivt', label: 'IVT max' },
  { key: 'fraud_threshold_block', label: 'Block max' },
];

function thresholdsOrdered(cfg: CampaignFraudConfig): boolean {
  return (
    cfg.fraud_threshold_pass <= cfg.fraud_threshold_suspect &&
    cfg.fraud_threshold_suspect <= cfg.fraud_threshold_ivt &&
    cfg.fraud_threshold_ivt <= cfg.fraud_threshold_block
  );
}

export function CampaignFraudSection({
  campaignId,
  canWrite,
  onCampaignFlagsChanged,
}: CampaignFraudSectionProps) {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [config, setConfig] = useState<CampaignFraudConfig | null>(null);
  const [draft, setDraft] = useState<CampaignFraudConfig | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [preview, setPreview] = useState<CampaignFraudPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [presets, setPresets] = useState<FraudPolicyPreset[]>([]);

  const presetOptions = useMemo(() => {
    if (!presets.length) return FRAUD_PRESET_OPTIONS;
    return presets.map((preset) => {
      const fallback = FRAUD_PRESET_OPTIONS.find((opt) => opt.id === preset.name);
      return {
        id: preset.name as FraudSensitivityPreset,
        label: fallback?.label ?? preset.name,
        description: fallback?.description ?? `pass<=${preset.pass}, block<=${preset.block}`,
      };
    });
  }, [presets]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [cfg, presetRows] = await Promise.all([
        fetchCampaignFraudConfig(campaignId),
        fetchFraudPresets(),
      ]);
      setPresets(presetRows);
      setConfig(cfg);
      setDraft(cfg);
      setPreview(null);
      setPreviewError(null);
    } catch (err) {
      setError(mapServiceError(err).message);
    } finally {
      setLoading(false);
    }
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  const bandRows = useMemo(() => {
    if (!draft) return [];
    return fraudTierBandRowsFromThresholds(
      draft.fraud_threshold_pass,
      draft.fraud_threshold_suspect,
      draft.fraud_threshold_ivt
    );
  }, [draft]);

  const validationError = useMemo(() => {
    if (!draft) return null;
    if (!thresholdsOrdered(draft)) {
      return 'Thresholds must be ordered: pass <= suspect <= ivt <= block (0-100).';
    }
    return null;
  }, [draft]);

  const applyPreset = (preset: FraudSensitivityPreset) => {
    if (!canWrite || !draft) return;
    void save({ preset });
  };

  const save = async (override?: { preset?: FraudSensitivityPreset }) => {
    if (!canWrite || !draft) return;
    if (!override && validationError) {
      pushToastMessage({ title: 'Invalid thresholds', message: validationError });
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const body = override?.preset
        ? { preset: override.preset, silent_reject_enabled: draft.silent_reject_enabled }
        : {
            fraud_threshold_pass: draft.fraud_threshold_pass,
            fraud_threshold_suspect: draft.fraud_threshold_suspect,
            fraud_threshold_ivt: draft.fraud_threshold_ivt,
            fraud_threshold_block: draft.fraud_threshold_block,
            silent_reject_enabled: draft.silent_reject_enabled,
          };
      const updated = await patchCampaignFraudConfig(campaignId, body);
      if (updated) {
        setConfig(updated);
        setDraft(updated);
        pushToastMessage({
          title: 'Fraud settings saved',
          message: 'Thresholds updated for this campaign.',
        });
        if (
          override?.preset === 'enhanced_defense' ||
          override?.preset === 'social_in_app'
        ) {
          onCampaignFlagsChanged?.();
        }
      }
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      const message = mapServiceError(err).message;
      setError(message);
      pushToastMessage({ title: 'Save failed', message });
    } finally {
      setSaving(false);
    }
  };

  const setThreshold = (key: ThresholdField, raw: string) => {
    if (!draft) return;
    const value = Math.max(0, Math.min(100, Number.parseInt(raw, 10) || 0));
    setDraft({ ...draft, [key]: value });
    setPreview(null);
  };

  const runPreview = async () => {
    if (!draft || validationError) {
      if (validationError) {
        pushToastMessage({ title: 'Invalid thresholds', message: validationError });
      }
      return;
    }
    setPreviewing(true);
    setPreviewError(null);
    try {
      const out = await previewCampaignFraudImpact(campaignId, {
        fraud_threshold_pass: draft.fraud_threshold_pass,
        fraud_threshold_suspect: draft.fraud_threshold_suspect,
        fraud_threshold_ivt: draft.fraud_threshold_ivt,
        fraud_threshold_block: draft.fraud_threshold_block,
      });
      setPreview(out);
    } catch (err) {
      const message = mapServiceError(err).message;
      setPreviewError(message);
      setPreview(null);
    } finally {
      setPreviewing(false);
    }
  };

  if (loading) {
    return <p className="loading-hint">Loading fraud settings...</p>;
  }

  if (!draft) {
    return <p className="text-muted text-sm">Fraud settings unavailable.</p>;
  }

  return (
    <div className="stack stack--lg" data-testid="campaign-fraud-section">
      <p className="text-muted text-sm">
        Adjust ML fraud score bands for this campaign. Changes apply on fraud-scorer within about 60
        seconds.
      </p>

      {error ? <p className="text-danger text-sm">{error}</p> : null}

      <section className="stack">
        <h3 className="subsection-title">Sensitivity preset</h3>
        <div className="button-row">
          {presetOptions.map((opt) => (
            <Button
              key={opt.id}
              label={opt.label}
              variant="secondary"
              size="sm"
              disabled={!canWrite || saving}
              onClick={() => applyPreset(opt.id)}
              data-testid={`fraud-preset-${opt.id}`}
            />
          ))}
        </div>
        <ul className="text-muted text-xs">
          {presetOptions.map((opt) => (
            <li key={`${opt.id}-desc`}>
              <strong>{opt.label}:</strong> {opt.description}
            </li>
          ))}
        </ul>
      </section>

      <section className="stack">
        <h3 className="subsection-title">Tier thresholds</h3>
        <div className="form-grid form-grid--2">
          {THRESHOLD_FIELDS.map((field) => (
            <label key={field.key} className="form-field">
              <span className="form-field__label">{field.label}</span>
              <input
                className="input font-mono"
                type="number"
                min={0}
                max={100}
                value={draft[field.key]}
                disabled={!canWrite || saving}
                onChange={(e) => setThreshold(field.key, e.target.value)}
                data-testid={`fraud-threshold-${field.key}`}
              />
            </label>
          ))}
        </div>
        {validationError ? <p className="text-danger text-sm">{validationError}</p> : null}
      </section>

      <section className="stack" data-testid="fraud-preview-panel">
        <h3 className="subsection-title">Impact preview (7 days)</h3>
        <p className="text-muted text-xs">
          Estimate how many unique IPs would leave the pass tier under the draft thresholds before
          you save.
        </p>
        <div className="button-row">
          <Button
            label={previewing ? 'Previewing...' : 'Preview impact'}
            variant="secondary"
            size="sm"
            disabled={previewing || saving || Boolean(validationError)}
            onClick={() => void runPreview()}
            data-testid="fraud-preview-impact"
          />
        </div>
        {previewError ? <p className="text-danger text-sm">{previewError}</p> : null}
        {preview ? (
          <div className="stack stack--sm">
            <p className="text-sm">
              <strong>{preview.affected_ips_7d}</strong> affected IPs (sample{' '}
              <span className="font-mono">{preview.sample_size}</span>)
            </p>
            <ul className="text-sm">
              <li>
                Suspect: <span className="font-mono">{preview.by_tier.suspect}</span>
              </li>
              <li>
                IVT: <span className="font-mono">{preview.by_tier.ivt}</span>
              </li>
              <li>
                Block: <span className="font-mono">{preview.by_tier.block}</span>
              </li>
            </ul>
            <p className="text-muted text-xs">{preview.disclaimer}</p>
          </div>
        ) : null}
      </section>

      <section className="stack">
        <h3 className="subsection-title">Actions by tier</h3>
        <div className="table-wrapper">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">Tier</th>
                <th scope="col">Score</th>
                <th scope="col">Action</th>
              </tr>
            </thead>
            <tbody>
              {bandRows.map((row) => (
                <tr key={row.tier}>
                  <td>{row.tier}</td>
                  <td className="font-mono">{row.range}</td>
                  <td>{row.action}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="stack">
        <Checkbox
          checked={draft.silent_reject_enabled}
          disabled={!canWrite || saving}
          label="Silent reject"
          onChange={(checked) => setDraft({ ...draft, silent_reject_enabled: checked })}
        />
        <p className="text-muted text-xs">
          On L1 fraud: return success to the client but skip budget and postbacks. Off returns HTTP 403.
          Enhanced defense preset turns this on and uses redirect decoys on clicks.
        </p>
      </section>

      {canWrite ? (
        <div className="button-row">
          <Button
            label={saving ? 'Saving...' : 'Save thresholds'}
            variant="primary"
            disabled={saving || Boolean(validationError)}
            onClick={() => void save()}
            data-testid="fraud-save-thresholds"
          />
          <Button
            label="Reset"
            variant="secondary"
            disabled={saving || !config}
            onClick={() => setDraft(config)}
          />
        </div>
      ) : null}
    </div>
  );
}
