/**
 * Shared control height contract for admin UI.
 * Height and radius come from CSS (--admin-control-height, --admin-radius-sm) via .admin-btn / .admin-select.
 * Do not set h-7/h-8/h-9 on Button, SelectTrigger, or Input in domains/shell (ui_slop.sh).
 */
export const ADMIN_CONTROL_HEIGHT_PX = 32;

export const adminControlClassNames = {
  button: 'admin-btn',
  select: 'admin-select',
  input: 'admin-input',
} as const;
