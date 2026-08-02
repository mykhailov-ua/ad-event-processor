import { el } from '../lib/dom.js';

/**
 * @param {{ onActivate: () => void, id?: string, className?: string, cells: HTMLElement[] }} opts
 */
export function clickableRow(opts) {
  const row = el('tr', {
    id: opts.id,
    className: opts.className,
    tabIndex: '0',
    onClick: () => opts.onActivate(),
    onKeydown: (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        opts.onActivate();
      }
    },
  }, opts.cells);
  return row;
}
