import { el } from '../lib/dom.js';

/**
 * Render a horizontal tab bar for switching panel sections.
 *
 * @param {{ tabs: Array<{ id: string, label: string }>, active: string, onChange: (id: string) => void }} opts
 * @returns {HTMLElement}
 */
export function renderTabBar(opts) {
  return el('div', { className: 'tab-bar', role: 'tablist' },
    opts.tabs.map((tab) =>
      el('button', {
        type: 'button',
        role: 'tab',
        'aria-selected': opts.active === tab.id ? 'true' : 'false',
        className: 'tab-bar__item' + (opts.active === tab.id ? ' tab-bar__item--active' : ''),
        onClick: () => opts.onChange(tab.id),
      }, tab.label),
    ),
  );
}
