/**
 * Shared layout surfaces (padding, border, radius) live in app.css @layer components.
 * Components pick a base + tone modifier; domain code should not copy these strings.
 */
export const uiSurfaces = {
  message: 'ui-message-surface',
  messageError: 'ui-message-surface ui-message-surface-error',
  messageMuted: 'ui-message-surface ui-message-surface-muted',
  messageSuccess: 'ui-message-surface ui-message-surface-success',
  messageWarning: 'ui-message-surface ui-message-surface-warning',
  panel: 'ui-panel-surface',
  control: 'ui-control-surface',
  toolbarBand: 'ui-toolbar-band',
  toolbarBandSplit: 'ui-toolbar-band ui-toolbar-band-split',
  toolbarBandActions: 'ui-toolbar-band-actions',
  statusMetricsBand: 'ui-status-metrics-band',
  chipRow: 'ui-chip-row',
  chip: 'ui-chip-surface',
  chipCount: 'ui-chip-count',
  summaryBand: 'ui-summary-band',
  summaryBandDivider: 'ui-summary-band-divider',
  tableHost: 'ui-table-host',
  tableHostFill: 'ui-table-host ui-table-host-fill',
  tableHostScroll: 'ui-table-host-scroll',
  directoryStack: 'ui-directory-stack',
  metaLinksBand: 'ui-meta-links-band',
  actionLinksBand: 'ui-action-links-band',
} as const;

export type UiMessageSurfaceTone = 'error' | 'muted' | 'success' | 'warning';

const messageToneClass: Record<UiMessageSurfaceTone, string> = {
  error: uiSurfaces.messageError,
  muted: uiSurfaces.messageMuted,
  success: uiSurfaces.messageSuccess,
  warning: uiSurfaces.messageWarning,
};

export function uiMessageSurfaceClass(tone: UiMessageSurfaceTone): string {
  return messageToneClass[tone];
}
