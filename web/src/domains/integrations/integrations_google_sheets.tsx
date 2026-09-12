import { FileSpreadsheet } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import type { useIntegrationsGoogleSheetsPageWorkspace } from '@/domains/integrations/use_integrations_google_sheets_page_workspace';
import { adminTypography } from '@/lib/admin_kit';
import { adminSpacing } from '@/lib/admin_spacing';
import { BentoIconBadge, BentoSection } from '@/shell/bento_card';
import { StubBanner } from '@/shell/stub_banner';

export type IntegrationsGoogleSheetsProps = ReturnType<
  typeof useIntegrationsGoogleSheetsPageWorkspace
>;

export function IntegrationsGoogleSheets({
  status,
  fetching,
  error,
  hasSnapshot,
  disconnecting,
  disconnectError,
  onConnect,
  onDisconnect,
}: IntegrationsGoogleSheetsProps) {
  const notConfigured = status?.message === 'not configured';
  const connected = status?.connected === true;
  const busy = fetching || disconnecting;

  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load Google Sheets status"
      fetchState={{ error, fetching, hasSnapshot }}
      title="Google Sheets"
    >
      {disconnectError ? integrationsPanelError(disconnectError, 'Disconnect failed') : null}

      {notConfigured ? (
        <StubBanner
          message="Set GOOGLE_SHEETS_CLIENT_ID and GOOGLE_SHEETS_CLIENT_SECRET on the control plane before connecting."
          title="Google Sheets not configured"
        />
      ) : null}

      <BentoSection data-testid="google-sheets-connection" title="Connection">
        <div className={`grid ${adminSpacing.gap.lg}`}>
          <div className={`flex flex-wrap items-center ${adminSpacing.gap.md}`}>
            <BentoIconBadge icon={FileSpreadsheet} tone="brand" />
            <div className={adminSpacing.flex.columnMd} data-role="connection-status">
              <span className={adminTypography.label}>Export destination</span>
              {connected ? (
                <Badge variant="secondary">Connected</Badge>
              ) : (
                <Badge variant="outline">Not connected</Badge>
              )}
            </div>
          </div>
          <p className={adminTypography.bodyMuted}>
            Connect your Google account to push report export rows into a spreadsheet from Export
            Hub. OAuth is per operator; refresh tokens are stored encrypted on the server.
          </p>
          {connected && status?.account_email ? (
            <p className={adminTypography.bodyMuted}>Signed in as {status.account_email}.</p>
          ) : null}
          {!connected && status?.message && status.message !== 'not configured' ? (
            <p className={adminTypography.bodyMuted}>{status.message}</p>
          ) : null}
          <div
            aria-label="Google Sheets connection actions"
            className={adminSpacing.flex.buttonGroup}
            data-testid="google-sheets-actions"
          >
            {connected ? (
              <Button
                data-testid="google-sheets-disconnect"
                disabled={busy}
                type="button"
                variant="destructive"
                onClick={() => void onDisconnect()}
              >
                Disconnect
              </Button>
            ) : (
              <Button
                data-testid="google-sheets-connect"
                disabled={busy || notConfigured}
                type="button"
                onClick={onConnect}
              >
                Connect Google account
              </Button>
            )}
          </div>
        </div>
      </BentoSection>
    </IntegrationsPageWithLoad>
  );
}
