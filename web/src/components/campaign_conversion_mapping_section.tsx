import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import {
  fetchAffiliateStatusPresets,
  fetchConversionMappings,
  replaceConversionMappings,
  type AffiliateStatusPreset,
  type ConversionMappingRow,
} from '../helpers/conversion_mapping_api.js';
import { ErrorBlock } from './error_block.js';
import { SectionCard } from './section_card.js';

export type CampaignConversionMappingSectionProps = {
  campaignId: string;
  canWrite?: boolean;
};

const EMPTY_ROW: ConversionMappingRow = {
  inbound_status: '',
  goal_name: '',
  payout_micro: 0,
};

/**
 * Campaign Integration tab panel for affiliate status to payout mapping.
 */
export function CampaignConversionMappingSection({
  campaignId,
  canWrite = false,
}: CampaignConversionMappingSectionProps) {
  const [rows, setRows] = useState<ConversionMappingRow[]>([]);
  const [presets, setPresets] = useState<AffiliateStatusPreset[]>([]);
  const [presetName, setPresetName] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [savedMsg, setSavedMsg] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    const [mappingRes, presetRes] = await Promise.all([
      to(fetchConversionMappings(campaignId)),
      to(fetchAffiliateStatusPresets()),
    ]);
    setLoading(false);
    const [mappingData, mappingErr] = mappingRes;
    const [presetData, presetErr] = presetRes;
    if (mappingErr) {
      setError(mappingErr.message);
      return;
    }
    setRows(mappingData && mappingData.length > 0 ? mappingData : [{ ...EMPTY_ROW }]);
    if (!presetErr && presetData) {
      setPresets(presetData);
      if (presetData.length > 0) {
        setPresetName(presetData[0]?.name ?? '');
      }
    }
  }, [campaignId]);

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * Apply a bundled affiliate status preset as editable mapping rows.
   */
  const applyPreset = () => {
    const preset = presets.find((p) => p.name === presetName);
    if (!preset) {
      return;
    }
    setRows(
      preset.statuses.map((s) => ({
        inbound_status: s.inbound_status,
        goal_name: s.goal_name,
        payout_micro: 0,
      }))
    );
    setSavedMsg('');
  };

  /**
   * Persist mapping table to the control plane.
   */
  const save = async () => {
    setSaving(true);
    setError('');
    setSavedMsg('');
    const cleaned = rows
      .map((r) => ({
        inbound_status: r.inbound_status.trim(),
        goal_name: r.goal_name.trim(),
        payout_micro: Number(r.payout_micro) || 0,
      }))
      .filter((r) => r.inbound_status !== '');
    const res = await to(replaceConversionMappings(campaignId, cleaned));
    setSaving(false);
    const [savedRows, saveErr] = res;
    setSaving(false);
    if (saveErr) {
      setError(saveErr.message);
      return;
    }
    setRows(savedRows && savedRows.length > 0 ? savedRows : [{ ...EMPTY_ROW }]);
    setSavedMsg('Mappings saved');
  };

  return (
    <SectionCard
      icon="coins"
      title="Conversion type payout"
      desc="Map inbound affiliate postback status to goal_name and revenue_micro in conversion reports."
    >
      {loading ? <p className="text-muted text-sm">Loading mappings...</p> : null}
      {error ? <ErrorBlock error={error} /> : null}
      {presets.length > 0 ? (
        <div className="form-row" style={{ marginBottom: '1rem' }}>
          <label className="form-label" htmlFor="conversion-mapping-preset">
            Preset from affiliate schema
          </label>
          <div className="form-inline">
            <select
              id="conversion-mapping-preset"
              className="input"
              value={presetName}
              onChange={(e) => setPresetName(e.target.value)}
              disabled={!canWrite}
            >
              {presets.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name}
                </option>
              ))}
            </select>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={!canWrite}
              onClick={applyPreset}
            >
              Load preset
            </button>
          </div>
        </div>
      ) : null}
      <table className="data-table data-table--compact" data-testid="conversion-mapping-table">
        <thead>
          <tr>
            <th>Inbound status</th>
            <th>Goal name</th>
            <th>Payout (micro)</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, idx) => (
            <tr key={`${row.inbound_status}-${idx}`}>
              <td>
                <input
                  className="input input--compact"
                  value={row.inbound_status}
                  disabled={!canWrite}
                  onChange={(e) => {
                    const next = [...rows];
                    next[idx] = { ...row, inbound_status: e.target.value };
                    setRows(next);
                  }}
                  placeholder="lead"
                />
              </td>
              <td>
                <input
                  className="input input--compact"
                  value={row.goal_name}
                  disabled={!canWrite}
                  onChange={(e) => {
                    const next = [...rows];
                    next[idx] = { ...row, goal_name: e.target.value };
                    setRows(next);
                  }}
                  placeholder="lead"
                />
              </td>
              <td>
                <input
                  className="input input--compact"
                  type="number"
                  min={0}
                  value={row.payout_micro}
                  disabled={!canWrite}
                  onChange={(e) => {
                    const next = [...rows];
                    next[idx] = { ...row, payout_micro: Number(e.target.value) };
                    setRows(next);
                  }}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {canWrite ? (
        <div className="form-inline" style={{ marginTop: '0.75rem' }}>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => setRows([...rows, { ...EMPTY_ROW }])}
          >
            Add row
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={saving}
            onClick={() => void save()}
          >
            {saving ? 'Saving...' : 'Save mappings'}
          </button>
          {savedMsg ? <span className="text-success text-sm">{savedMsg}</span> : null}
        </div>
      ) : null}
      <p className="text-muted text-sm" style={{ marginTop: '0.75rem' }}>
        Unmapped inbound statuses produce zero payout in the{' '}
        <Link to="/reports/conversion-type-payout">conversion type payout</Link> report. Payout
        applies on cold-path CH ingest when conversion payload includes{' '}
        <code className="code-inline">status</code>.
      </p>
    </SectionCard>
  );
}
