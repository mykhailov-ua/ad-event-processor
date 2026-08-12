import { ICON_PATHS } from './icon_paths.js';

export type IconOpts = {
  size?: number;
  className?: string;
  /** Ignored (Lucide stroke legacy); kept for call-site compatibility. */
  strokeWidth?: number;
};

/**
 * Render a Phosphor regular icon as inline SVG (256 viewBox, fill).
 */
export function renderIcon(name: string, opts: IconOpts = {}): SVGElement | null {
  const tags = ICON_PATHS[name];
  if (!tags?.length) return null;

  const size = opts.size ?? 16;
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
  svg.setAttribute('width', '1em');
  svg.setAttribute('height', '1em');
  svg.style.width = `${size / 16}em`;
  svg.style.height = `${size / 16}em`;
  svg.setAttribute('viewBox', '0 0 256 256');
  svg.setAttribute('fill', 'currentColor');
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
    const tagName = m[1];
    const attrs = m[2] ?? '';
    const node = document.createElementNS('http://www.w3.org/2000/svg', tagName);
    const attrRe = /([\w:-]+)="([^"]*)"/g;
    let match: RegExpExecArray | null;
    while ((match = attrRe.exec(attrs)) !== null) {
      node.setAttribute(match[1], match[2]);
    }
    if (!node.hasAttribute('fill')) {
      node.setAttribute('fill', 'currentColor');
    }
    svg.appendChild(node);
  }

  return svg;
}
