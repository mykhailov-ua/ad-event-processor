/**
 * Shared control height contract for admin UI.
 * Use h-8 (32px) on Button, SelectTrigger, and Input in domains/shell.
 */
export const ADMIN_CONTROL_HEIGHT_PX = 32;

export const adminControlClassNames = {
  button: 'h-8',
  select: 'h-8',
  input: 'h-8',
} as const;
