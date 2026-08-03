/**
 * Mask a Telegram user identifier for admin UI (hash prefix only).
 *
 * @param {string|null|undefined} value Raw tg_user_id or SHA256 hex
 * @param {{ prefixLen?: number }} [opts]
 * @returns {string}
 */
export function maskTelegramUserId(value, opts = {}) {
  const prefixLen = opts.prefixLen ?? 8;
  const raw = String(value ?? '').trim();
  if (!raw) return '—';
  if (/^\d+$/.test(raw)) {
    return `tg:…${raw.slice(-4)}`;
  }
  if (raw.length <= prefixLen + 3) {
    return raw;
  }
  return `${raw.slice(0, prefixLen)}…`;
}

/**
 * Short privacy note for Telegram analytics views.
 *
 * @returns {string}
 */
export function telegramPiiNotice() {
  return 'Analytics use hashed tg_user_id only; raw Telegram IDs are not shown.';
}
