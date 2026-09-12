import { Link, useLocation } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { adminSpacing } from '@/lib/admin_spacing';
import { reportTitleFromKey } from '@/lib/report_paths';
import { cn } from '@/lib/utils';
import { CustomerScopeGate } from '@/shell/customer_scope_gate';
import {
  buildExportOnlyReportHubHref,
  type ExportOnlyReportHubHrefParams,
} from '@/shell/export_only_report_hub_href';
import { exportOnlyReportStubBannerMessage } from '@/shell/export_only_report_stub_message';
import { PageChrome } from '@/shell/page_chrome';
import { StubBanner } from '@/shell/stub_banner';

export type ExportOnlyReportStubProps = {
  reportKey: string;
  title?: string;
  customerId?: string;
  from?: string;
  to?: string;
  format?: string;
  defaultRange?: string;
  requiresCustomer?: boolean;
  catalogUnknown?: boolean;
};

export function ExportOnlyReportStub({
  reportKey,
  title,
  customerId,
  from,
  to,
  format,
  defaultRange = '7d',
  requiresCustomer = false,
  catalogUnknown = false,
}: ExportOnlyReportStubProps) {
  const location = useLocation();
  const resolvedTitle = title?.trim() || reportTitleFromKey(reportKey);
  const trimmedCustomerId = customerId?.trim() ?? '';

  const hubParams: ExportOnlyReportHubHrefParams = {
    reportKey,
    customerId: trimmedCustomerId || undefined,
    from,
    to,
    format,
    defaultRange,
  };
  const exportHubHref = buildExportOnlyReportHubHref(
    hubParams,
    `${location.pathname}${location.search}`
  );

  const exportContent = (
    <>
      <StubBanner
        message={exportOnlyReportStubBannerMessage(catalogUnknown)}
        title="Control Plane export mode"
      />
      <div className={adminSpacing.flex.buttonGroup}>
        <Button asChild type="button" variant="brand">
          <Link to={exportHubHref}>Open Export Hub</Link>
        </Button>
      </div>
    </>
  );

  return (
    <PageChrome title={resolvedTitle}>
      <div className={cn('grid min-w-0', adminSpacing.gap.xl)}>
        {requiresCustomer ? (
          <CustomerScopeGate
            customerId={trimmedCustomerId}
            description="Add customer_id to the URL before opening Export Hub for this report."
            testId="export-only-report-scope"
            title="Customer required"
          >
            {exportContent}
          </CustomerScopeGate>
        ) : (
          exportContent
        )}
      </div>
    </PageChrome>
  );
}
