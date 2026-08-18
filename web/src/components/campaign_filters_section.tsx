import { useState } from 'react';
import { to } from '../lib/to.js';
import { patchCampaign } from '../helpers/campaign_admin_api.js';
import {
  emptyTrafficFilterRules,
  parseTrafficFilter,
  serializeTrafficFilter,
} from '../helpers/traffic_filter.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { Button } from './button.js';
import { Checkbox } from './checkbox.js';

export type CampaignFiltersSectionProps = {
  campaignId: string;
  referrerFilter: string;
  canWrite: boolean;
  onSaved?: () => void;
};

export function CampaignFiltersSection({
  campaignId,
  referrerFilter,
  canWrite,
  onSaved,
}: CampaignFiltersSectionProps) {
  const initial = parseTrafficFilter(referrerFilter);
  const [allowInput, setAllowInput] = useState(initial.allowReferrers.join(', '));
  const [blockInput, setBlockInput] = useState(initial.blockReferrers.join(', '));
  const [blockEmpty, setBlockEmpty] = useState(initial.blockEmptyReferrer);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async () => {
    if (!canWrite) return;
    setSaving(true);
    setError(null);
    const next = {
      allowReferrers: allowInput
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      blockReferrers: blockInput
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      blockEmptyReferrer: blockEmpty,
    };
    const [, err] = await to(
      patchCampaign(campaignId, {
        referrer_filter: serializeTrafficFilter(next),
      })
    );
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(mapServiceError(err).message);
      return;
    }
    pushToastMessage({ title: 'Filters saved', message: 'Traffic rules updated' });
    onSaved?.();
  };

  const reset = () => {
    const empty = emptyTrafficFilterRules();
    setAllowInput('');
    setBlockInput('');
    setBlockEmpty(empty.blockEmptyReferrer);
    setError(null);
  };

  return (
    <div className="section-card stack">
      <h3 className="subsection-title">Traffic filters</h3>
      <p className="text-muted text-sm">
        Structured referrer rules stored as JSON. Hot path reads campaign config after publish.
      </p>
      {error ? <p className="text-danger text-sm">{error}</p> : null}
      <label className="form-field" htmlFor="flt-allow">
        Allow referrers (comma-separated hostnames)
        <input
          id="flt-allow"
          className="form-input"
          disabled={!canWrite}
          value={allowInput}
          onChange={(e) => setAllowInput(e.target.value)}
        />
      </label>
      <label className="form-field" htmlFor="flt-block">
        Block referrers
        <input
          id="flt-block"
          className="form-input"
          disabled={!canWrite}
          value={blockInput}
          onChange={(e) => setBlockInput(e.target.value)}
        />
      </label>
      <Checkbox
        label="Block empty referrer"
        checked={blockEmpty}
        disabled={!canWrite}
        onChange={setBlockEmpty}
      />
      {canWrite ? (
        <div className="cluster--actions">
          <Button
            label={saving ? 'Saving…' : 'Save filters'}
            variant="primary"
            size="sm"
            loading={saving}
            disabled={saving}
            onClick={() => void save()}
          />
          <Button label="Reset" variant="secondary" size="sm" disabled={saving} onClick={reset} />
        </div>
      ) : null}
    </div>
  );
}
