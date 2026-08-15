import { useCallback } from 'react';
import { mountOpsHomePanel } from '../../panels/ops_home_panel.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Operations home — imperative panel with live feed cleanup on unmount.
 */
export function OpsHomePage() {
  const mount = useCallback((host: HTMLElement) => mountOpsHomePanel(host), []);
  return <LegacyPanelHost active mount={mount} deps={[]} />;
}
