import { panelError } from '@/shell/panel_error';

export function billingPanelError(error: Error, title: string) {
  return panelError(error, title);
}
