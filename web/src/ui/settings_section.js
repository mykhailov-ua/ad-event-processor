import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

/**
 * @param {string} title
 * @param {string} [description]
 * @param {string} [icon]
 * @param {...(HTMLElement|null|false|undefined)} children
 */
export function renderSettingsSection(title, description, icon, ...children) {
  const titleRow = el('div', { className: 'settings-panel__title-row' },
    icon ? renderIcon(icon, { size: 18, className: 'settings-panel__icon' }) : null,
    el('h2', { className: 'settings-panel__title' }, title),
  );

  return el('section', { className: 'settings-panel' },
    el('div', { className: 'settings-panel__header' },
      titleRow,
      description
        ? el('p', { className: 'settings-panel__desc' }, description)
        : null,
    ),
    el('div', { className: 'settings-panel__body' }, ...children),
  );
}

/**
 * @param {string} label
 * @param {string|Node} value
 */
export function renderSettingsSummaryItem(label, value) {
  return el('div', { className: 'settings-summary__item' },
    el('div', { className: 'settings-summary__label' }, label),
    el('div', { className: 'settings-summary__value' }, value),
  );
}
