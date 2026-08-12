import { el, replaceChildren } from '../lib/dom.js';
import { mount as mountChipRow } from './chip_row.js';
import { debounce } from '../helpers/debounce.js';

export type FilterChipItem = {
  value: string;
  label: string;
};

export type FilterToolbarOpts = {
  search?: boolean;
  searchPlaceholder?: string;
  searchValue?: string;
  onSearch?: ((value: string) => void) | null;
  searchDebounceMs?: number;
  chips?: FilterChipItem[];
  chipSelected?: string;
  onChipSelect?: ((value: string) => void) | null;
  actions?: HTMLElement[];
  /** Prev/next control — rendered in the toolbar actions slot (right side of the plate). */
  pagination?: HTMLElement | null;
  leading?: HTMLElement[];
};

export type FilterToolbarHandle = {
  destroy: () => void;
  rerender: () => void;
};

/**
 * Mount a filter toolbar with search, chips, and action slots.
 */
export function mountFilterToolbar(container: HTMLElement, opts: FilterToolbarOpts): FilterToolbarHandle {
  let destroyed = false;
  const debounceMs = opts.searchDebounceMs ?? 400;
  let pendingSearch = opts.searchValue ?? '';
  const flushSearch = opts.onSearch
    ? debounce(() => { opts.onSearch?.(pendingSearch); }, debounceMs)
    : null;

  function render(): void {
    if (destroyed) return;

    const children: HTMLElement[] = [];

    const leading = opts.leading ?? [];
    for (const node of leading) {
      if (node) children.push(el('div', { className: 'filter-toolbar__leading' }, node));
    }

    if (opts.search) {
      children.push(
        el('div', { className: 'filter-toolbar__search' },
          el('input', {
            type: 'search',
            className: 'form-input form-input--sm filter-toolbar__search-input',
            placeholder: opts.searchPlaceholder ?? 'Search…',
            value: opts.searchValue ?? '',
            'aria-label': opts.searchPlaceholder ?? 'Search',
            onInput: (e: Event) => {
              const value = (e.target as HTMLInputElement).value;
              pendingSearch = value;
              if (flushSearch) flushSearch();
              else opts.onSearch?.(value);
            },
          }),
        ),
      );
    }

    if (opts.chips?.length) {
      const chipWrap = el('div', { className: 'filter-toolbar__chips' });
      mountChipRow(chipWrap, {
        items: opts.chips,
        selected: opts.chipSelected ?? '',
        onSelect: (v) => opts.onChipSelect?.(v),
      });
      children.push(chipWrap);
    }

    const spacer = el('div', { className: 'filter-toolbar__spacer' });

    const actionNodes: HTMLElement[] = [];
    if (opts.pagination) actionNodes.push(opts.pagination);
    const extraActions = opts.actions?.filter(Boolean) ?? [];
    actionNodes.push(...extraActions);

    const actionsEl = actionNodes.length
      ? el('div', { className: 'filter-toolbar__actions' }, ...actionNodes)
      : null;

    replaceChildren(
      container,
      el('div', { className: 'filter-toolbar elevation-sunken' }, ...children, spacer, actionsEl),
    );
  }

  render();

  return {
    destroy() {
      destroyed = true;
      container.replaceChildren();
    },
    rerender: render,
  };
}
