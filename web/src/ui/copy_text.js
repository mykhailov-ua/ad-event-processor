import { el } from '../lib/dom.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderIcon } from './icon.js';

/**
 * Copy text to clipboard with instant visual feedback.
 *
 * @param {string} text
 * @param {string} [label]
 * @returns {Promise<boolean>}
 */
export async function copyToClipboard(text, label = 'Copied to clipboard') {
  if (!text) return false;
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      textarea.remove();
    }
    pushToastMessage({ title: 'Copied!', message: label });
    return true;
  } catch (e) {
    pushToastMessage({ title: 'Copy failed', message: 'Could not access clipboard' });
    return false;
  }
}

/**
 * Render a clickable, copyable element (UUID, key, code, URL, IP).
 * Shows copy icon on hover, copies on click, and displays checkmark animation.
 *
 * @param {string} text - Full value to copy
 * @param {string|HTMLElement} [displayContent] - Optional custom display content
 * @param {{ className?: string, title?: string, showIcon?: boolean }} [opts]
 * @returns {HTMLElement}
 */
export function renderCopyable(text, displayContent, opts = {}) {
  if (!text) return el('span', { className: 'text-muted' }, '—');

  const content = displayContent ?? text;
  const showIcon = opts.showIcon !== false;

  const btn = el('button', {
    type: 'button',
    className: `copyable-btn ${opts.className ?? ''}`.trim(),
    title: opts.title ?? `Click to copy: ${text}`,
    'aria-label': `Copy ${text}`,
    onClick: async (e) => {
      e.stopPropagation();
      e.preventDefault();
      const success = await copyToClipboard(
        text,
        `Copied: ${text.length > 24 ? text.slice(0, 20) + '…' : text}`,
      );
      if (success) {
        btn.classList.add('copyable-btn--copied');
        setTimeout(() => btn.classList.remove('copyable-btn--copied'), 1500);
      }
    },
  },
    el('span', { className: 'copyable-btn__text' }, content),
    showIcon ? renderIcon('copy', { size: 13, className: 'copyable-btn__icon' }) : null,
    showIcon ? renderIcon('check', { size: 13, className: 'copyable-btn__check' }) : null,
  );

  return btn;
}

/**
 * Render a UUID with middle truncation that is clickable and copyable.
 *
 * @param {string} uuid
 * @param {{ className?: string }} [opts]
 * @returns {HTMLElement}
 */
export function renderCopyableUuid(uuid, opts = {}) {
  if (!uuid) return el('span', { className: 'text-muted' }, '—');
  const truncated = uuid.length > 16 ? `${uuid.slice(0, 8)}…${uuid.slice(-8)}` : uuid;
  const textNode = el('span', { className: 'font-mono text-hint' }, truncated);
  return renderCopyable(uuid, textNode, opts);
}
