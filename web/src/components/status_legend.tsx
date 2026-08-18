import { StatusBadge } from './status_badge.js';

export function CampaignStatusLegend() {
  return (
    <div className="status-legend" aria-label="Status legend">
      <span className="status-legend__label">Status</span>
      <StatusBadge status="ACTIVE" kind="campaign" />
      <StatusBadge status="PAUSED" kind="campaign" />
      <StatusBadge status="ARCHIVED" kind="campaign" />
    </div>
  );
}
