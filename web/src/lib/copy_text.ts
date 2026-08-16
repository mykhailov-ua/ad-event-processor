import { pushToastMessage } from '../helpers/toast_ui.js';

/**
 * Copy text to clipboard with instant visual feedback.
 */
export async function copyToClipboard(text: string, label = 'Copied to clipboard'): Promise<boolean> {
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
  } catch {
    pushToastMessage({ title: 'Copy failed', message: 'Could not access clipboard' });
    return false;
  }
}
