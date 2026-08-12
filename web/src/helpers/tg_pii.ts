export type MaskTelegramUserIdOpts = {
  prefixLen?: number;
};

/**
 * Mask a Telegram user identifier for admin UI (hash prefix only).
 */
export function maskTelegramUserId(
  value: string | null | undefined,
  opts: MaskTelegramUserIdOpts = {},
): string {
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
 */
export function telegramPiiNotice(): string {
  return 'Analytics use hashed tg_user_id only; raw Telegram IDs are not shown.';
}
