import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

type OptionalChild = HTMLElement | null | false | undefined;

/**
 * Build a section title row with optional icon.
 */
function renderTitleRow(title: string | Node, icon?: string): Node {
  if (typeof title !== 'string') return title;
  return el('div', { className: 'settings-panel__title-row' },
    icon ? renderIcon(icon, { size: 18, className: 'settings-panel__icon' }) : null,
    el('h2', { className: 'settings-panel__title' }, title),
  );
}

export type SectionCardProps = {
  title?: string | Node;
  desc?: string;
  icon?: string;
  urgent?: 'normal' | 'warning' | 'danger';
  children?: OptionalChild[] | HTMLElement | null | false | undefined;
  className?: string;
};

/**
 * Render a raised section surface with optional urgency border.
 */
export function renderSectionCard(props: SectionCardProps): HTMLElement {
  const urgentClass = props.urgent ? ` settings-panel--urgent-${props.urgent}` : '';
  const titleRow = props.title ? renderTitleRow(props.title, props.icon) : null;
  const header = titleRow || props.desc
    ? el('div', { className: 'settings-panel__header' },
        titleRow,
        props.desc
          ? el('p', { className: 'settings-panel__desc' }, props.desc)
          : null,
      )
    : null;

  const body = el('div', { className: 'settings-panel__body' },
    ...(Array.isArray(props.children) ? props.children : [props.children]).filter(Boolean),
  );

  return el('section', { className: `settings-panel${urgentClass} ${props.className ?? ''}`.trim() },
    header,
    body,
  );
}
