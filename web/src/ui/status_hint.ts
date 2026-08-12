import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

export type StatusHintProps = {
  tone: 'info' | 'error' | 'success';
  message: string | Node;
  icon?: string;
  className?: string;
};

/**
 * Render inline persistent feedback with tone-specific styling.
 */
export function renderStatusHint(props: StatusHintProps): HTMLElement {
  const tone = props.tone ?? 'info';
  const defaultIcon = tone === 'error' ? 'alert-circle' : tone === 'success' ? 'check-circle' : 'info';
  const iconName = props.icon ?? defaultIcon;

  return el('div', { className: `status-hint status-hint--${tone} ${props.className ?? ''}`.trim() },
    renderIcon(iconName, { size: 16, className: `status-hint__icon` }),
    el('div', { className: 'status-hint__message' }, props.message),
  );
}
