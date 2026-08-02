import { el } from '../lib/dom.js';

/**
 * Render a raised section surface with optional urgency border.
 *
 * @param {{ title?: string|Node, desc?: string, urgent?: 'normal'|'warning'|'danger', children?: Array<HTMLElement|null|false>|HTMLElement, className?: string }} props
 * @returns {HTMLElement}
 */
export function renderSectionCard(props) {
  const urgentClass = props.urgent ? ` section-card--urgent-${props.urgent}` : '';
  const header = props.title || props.desc
    ? el('div', { className: 'settings-panel__header' },
        props.title
          ? el('div', { className: 'settings-panel__title-row' },
              typeof props.title === 'string'
                ? el('h2', { className: 'settings-panel__title' }, props.title)
                : props.title
            )
          : null,
        props.desc
          ? el('p', { className: 'settings-panel__desc' }, props.desc)
          : null,
      )
    : null;

  const body = el('div', { className: 'settings-panel__body' },
    ...(Array.isArray(props.children) ? props.children : [props.children]).filter(Boolean)
  );

  return el('section', { className: `settings-panel${urgentClass} ${props.className ?? ''}`.trim() },
    header,
    body
  );
}
