import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

type OptionalChild = HTMLElement | null | false | undefined;

/**
 * Render a settings panel section with optional icon and description.
 */
export function renderSettingsSection(
  title: string,
  description?: string,
  icon?: string,
  ...children: OptionalChild[]
): HTMLElement {
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
 * Render a label/value row for settings summary grids.
 */
export function renderSettingsSummaryItem(label: string, value: string | Node): HTMLElement {
  return el('div', { className: 'settings-summary__item' },
    el('div', { className: 'settings-summary__label' }, label),
    el('div', { className: 'settings-summary__value' }, value),
  );
}
