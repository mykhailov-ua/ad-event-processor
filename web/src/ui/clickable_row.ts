import { el } from '../lib/dom.js';

export type ClickableRowOpts = {
  onActivate: () => void;
  id?: string;
  className?: string;
  cells: HTMLElement[];
};

/**
 * Render a keyboard-accessible table row that activates on click or Enter.
 */
export function clickableRow(opts: ClickableRowOpts): HTMLTableRowElement {
  const row = el('tr', {
    id: opts.id,
    className: opts.className,
    tabIndex: '0',
    onClick: () => opts.onActivate(),
    onKeydown: (e: Event) => {
      const ke = e as KeyboardEvent;
      if (ke.key === 'Enter' || ke.key === ' ') {
        ke.preventDefault();
        opts.onActivate();
      }
    },
  }, opts.cells);
  return row as HTMLTableRowElement;
}
