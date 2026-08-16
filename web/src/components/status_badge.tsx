import { displayLabel } from '../helpers/display_labels.js';
import { statusClassFor } from '../helpers/status.js';

export type StatusBadgeProps = {
  status: string;
  kind?: 'campaign' | 'service' | 'invoice';
  label?: string;
};

/**
 * Colored status badge for domain-specific status values.
 */
export function StatusBadge({ status, kind = 'campaign', label }: StatusBadgeProps) {
  const text = label ?? displayLabel(status);
  const mod = statusClassFor(status, kind);
  return <span className={`status-badge status-badge--${mod}`}>{text}</span>;
}
