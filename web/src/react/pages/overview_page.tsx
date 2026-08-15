import { useCallback } from 'react';
import { mountOverviewPanel } from '../../panels/overview_panel.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Operator / buyer home overview dashboard.
 */
export function OverviewPage() {
  const mount = useCallback((host: HTMLElement) => mountOverviewPanel(host), []);
  return <LegacyPanelHost active mount={mount} deps={[]} />;
}
