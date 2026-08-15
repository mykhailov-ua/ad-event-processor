import { useCallback } from 'react';
import { mountRtbIntegrationPanel } from '../../panels/rtb_integration_panel.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * RTB integration profile configuration.
 */
export function RtbIntegrationPage() {
  const mount = useCallback((host: HTMLElement) => mountRtbIntegrationPanel(host), []);
  return <LegacyPanelHost active mount={mount} deps={[]} />;
}
