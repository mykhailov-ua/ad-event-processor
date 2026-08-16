import { createElement, type CSSProperties, type ReactElement } from 'react';
import { ICON_PATHS } from '../lib/icon_paths.js';

export type IconProps = {
  name: string;
  size?: number;
  className?: string;
};

/**
 * Parse a static Phosphor SVG primitive tag into a React element.
 */
function parseIconElement(tag: string): ReactElement | null {
  const m = tag.match(/^<(\w+)\b([^>]*)\/?>$/);
  if (!m) return null;
  const tagName = m[1];
  const attrsStr = m[2] ?? '';
  const props: Record<string, string> = {};
  const attrRe = /([\w:-]+)="([^"]*)"/g;
  let match: RegExpExecArray | null;
  while ((match = attrRe.exec(attrsStr)) !== null) {
    props[match[1]] = match[2];
  }
  if (!props.fill) props.fill = 'currentColor';
  return createElement(tagName, props);
}

/**
 * Render a Phosphor regular icon as inline SVG.
 */
export function Icon({ name, size = 16, className }: IconProps) {
  const tags = ICON_PATHS[name];
  if (!tags?.length) return null;

  const style: CSSProperties = {
    width: `${size / 16}em`,
    height: `${size / 16}em`,
  };

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 256 256"
      fill="currentColor"
      aria-hidden="true"
      focusable="false"
      className={className ? `ui-icon ${className}` : 'ui-icon'}
      style={style}
      width="1em"
      height="1em"
    >
      {tags.map((tag, index) => (
        <IconPrimitive key={`${name}-${index}`} tag={tag} />
      ))}
    </svg>
  );
}

function IconPrimitive({ tag }: { tag: string }) {
  const el = parseIconElement(tag);
  return el ?? null;
}
