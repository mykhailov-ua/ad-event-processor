import { ICON_PATHS } from './icon_paths.js';

/**
 * Render a Lucide stroke icon as inline SVG.
 *
 * @param {string} name
 * @param {{ size?: number, className?: string, strokeWidth?: number }} [opts]
 * @returns {SVGElement|null}
 */
export function renderIcon(name, opts = {}) {
  const tags = ICON_PATHS[name];
  if (!tags?.length) return null;

  const size = opts.size ?? 16;
  const strokeWidth = opts.strokeWidth ?? 1.5;
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
  svg.setAttribute('width', '1em');
  svg.setAttribute('height', '1em');
  svg.style.width = `${size / 16}em`;
  svg.style.height = `${size / 16}em`;
  svg.setAttribute('viewBox', '0 0 24 24');
  svg.setAttribute('fill', 'none');
  svg.setAttribute('stroke', 'currentColor');
  svg.setAttribute('stroke-width', String(strokeWidth));
  svg.setAttribute('stroke-linecap', 'round');
  svg.setAttribute('stroke-linejoin', 'round');
  svg.setAttribute('aria-hidden', 'true');
  svg.setAttribute('focusable', 'false');
  if (opts.className) {
    svg.setAttribute('class', `ui-icon ${opts.className}`.trim());
  } else {
    svg.setAttribute('class', 'ui-icon');
  }

  for (const tag of tags) {
    const m = tag.match(/^<(\w+)\b([^>]*)\/?>$/);
    if (!m) continue;
    const [, tagName, attrs] = m;
    const el = document.createElementNS('http://www.w3.org/2000/svg', tagName);
    const attrRe = /([\w:-]+)="([^"]*)"/g;
    let match;
    while ((match = attrRe.exec(attrs)) !== null) {
      el.setAttribute(match[1], match[2]);
    }
    svg.appendChild(el);
  }

  return svg;
}
