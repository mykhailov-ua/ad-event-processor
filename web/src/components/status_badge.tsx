import { displayLabel } from '../helpers/display_labels.js';
import { statusClassFor } from '../helpers/status.js';

export type StatusBadgeProps = {
  status: string;
  kind?: 'campaign' | 'service' | 'invoice';
  label?: string;
};

export function StatusBadge({ status, kind = 'campaign', label }: StatusBadgeProps) {
  const text = label ?? displayLabel(status);
  const mod = statusClassFor(status, kind);
  return <span className={`status-badge status-badge--${mod}`}>{text}</span>;
}
