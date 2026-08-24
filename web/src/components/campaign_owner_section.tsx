import { useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { assignCampaignOwner, fetchTeamOverview } from '../helpers/team_api.js';
import type { TeamMemberDTO } from '../types/team.js';
import { SectionCard } from './section_card.js';

export type CampaignOwnerSectionProps = {
  campaignId: string;
  customerId: string;
  ownerUserId?: string;
  canWrite: boolean;
  onAssigned?: (userId: string) => void;
};

export function CampaignOwnerSection({
  campaignId,
  customerId,
  ownerUserId = '',
  canWrite,
  onAssigned,
}: CampaignOwnerSectionProps) {
  const [members, setMembers] = useState<TeamMemberDTO[]>([]);
  const [selected, setSelected] = useState(ownerUserId);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setSelected(ownerUserId);
  }, [ownerUserId]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      const [overview, err] = await to(fetchTeamOverview(customerId));
      if (cancelled) return;
      setLoading(false);
      if (err) {
        setMembers([]);
        return;
      }
      setMembers(overview?.members ?? []);
    })();
    return () => {
      cancelled = true;
    };
  }, [customerId]);

  const saveOwner = async (userId: string) => {
    if (!canWrite || saving || !userId) return;
    setSaving(true);
    const [, err] = await to(assignCampaignOwner(campaignId, userId));
    setSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) {
        setSelected(ownerUserId);
        return;
      }
      pushToastMessage({ title: 'Owner assign failed', message: mapServiceError(err).message });
      setSelected(ownerUserId);
      return;
    }
    pushToastMessage({ title: 'Owner updated', message: 'Campaign owner assigned' });
    onAssigned?.(userId);
  };

  if (!canWrite) return null;

  return (
    <div data-testid="campaign-owner-panel">
      <SectionCard
        icon="user"
        title="Campaign owner"
        desc="Media buyer accountable for spend and portfolio filters (CPA-M5)."
      >
        {loading ? <p className="text-muted text-sm">Loading team...</p> : null}
        {!loading && members.length === 0 ? (
          <p className="text-muted text-sm">No team members for this customer.</p>
        ) : null}
        {!loading && members.length > 0 ? (
          <label className="form-field" htmlFor="campaign-owner-select">
            Owner
            <select
              id="campaign-owner-select"
              className="form-input form-input--sm"
              value={selected}
              disabled={saving}
              data-testid="campaign-owner-select"
              onChange={(e) => {
                const next = e.target.value;
                setSelected(next);
                if (next) void saveOwner(next);
              }}
            >
              {!selected ? (
                <option value="" disabled>
                  Select owner...
                </option>
              ) : null}
              {members.map((m) => (
                <option key={m.user_id} value={m.user_id}>
                  {m.email} ({m.role})
                </option>
              ))}
            </select>
          </label>
        ) : null}
      </SectionCard>
    </div>
  );
}
