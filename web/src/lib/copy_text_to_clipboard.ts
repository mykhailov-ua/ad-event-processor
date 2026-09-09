// Clipboard helper for admin copy actions. Secure origins use navigator.clipboard;
// HTTP dev hosts fall back to textarea + document.execCommand('copy').
export async function copyTextToClipboard(text: string): Promise<void> {
  const trimmed = text.trim();
  if (trimmed === '') {
    throw new Error('copyTextToClipboard: empty value');
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(trimmed);
      return;
    } catch {
      // NotAllowedError or insecure context: fall through to execCommand.
    }
  }

  if (typeof document === 'undefined') {
    throw new Error('copyTextToClipboard: document unavailable');
  }

  const textarea = document.createElement('textarea');
  textarea.value = trimmed;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.left = '-9999px';
  textarea.style.top = '0';
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  textarea.setSelectionRange(0, trimmed.length);

  let copied = false;
  try {
    copied = document.execCommand('copy');
  } finally {
    document.body.removeChild(textarea);
  }

  if (!copied) {
    throw new Error('copyTextToClipboard: execCommand failed');
  }
}
