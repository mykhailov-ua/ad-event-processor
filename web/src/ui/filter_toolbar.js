import { el, replaceChildren } from '../lib/dom.js';
import { mount as mountChipRow } from './chip_row.js';
import { debounce } from '../helpers/debounce.js';

/**
 * @typedef {{ value: string, label: string }} ChipItem
 */

/**
 * @typedef {Object} FilterToolbarOpts
 * @property {boolean} [search]
 * @property {string} [searchPlaceholder]
 * @property {string} [searchValue]
 * @property {(value: string) => void} [onSearch]
 * @property {number} [searchDebounceMs]
 * @property {ChipItem[]} [chips]
 * @property {string} [chipSelected]
 * @property {(value: string) => void} [onChipSelect]
 * @property {HTMLElement[]} [actions]
 * @property {HTMLElement[]} [leading]
 */

/**
 * Mount a filter toolbar with search, chips, and action slots.
 *
 * @param {HTMLElement} container
 * @param {FilterToolbarOpts} opts
 * @returns {{ destroy: () => void, rerender: () => void }}
 */
export function mountFilterToolbar(container, opts) {
  let destroyed = false;
  const debounceMs = opts.searchDebounceMs ?? 400;
  const onSearchDebounced = opts.onSearch
    ? debounce((value) => opts.onSearch?.(value), debounceMs)
    : null;

  function render() {
    if (destroyed) return;

    const children = [];

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
            onInput: (e) => {
              const value = e.target.value;
              if (onSearchDebounced) onSearchDebounced(value);
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

    const actions = opts.actions?.filter(Boolean) ?? [];
    const actionsEl = actions.length
      ? el('div', { className: 'filter-toolbar__actions' }, ...actions)
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
