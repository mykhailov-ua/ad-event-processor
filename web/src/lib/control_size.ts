/**
 * Shared control height contract for admin UI.
 * Canonical token: admin_kit.controlHeight (`min-h-7`) via adminChrome.control.
 * Use default Button / Input / SelectTrigger in domains/shell -- no manual h-* overrides.
 */
export const ADMIN_CONTROL_HEIGHT_PX = 28;

export const adminControlClassNames = {
  button: 'min-h-7',
  select: 'min-h-7',
  input: 'min-h-7',
} as const;
