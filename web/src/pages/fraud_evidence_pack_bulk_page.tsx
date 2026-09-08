import { Link } from 'react-router-dom';

import { buildReportJobsHref } from '@/lib/report_paths';
import { PageLayout } from '@/shell/page_layout';
import { StubBanner } from '@/shell/stub_banner';
import { DirectoryStack, MetaLinksBand } from '@/shell/ui_bands';

export function FraudEvidencePackBulkPage() {
  const jobsHref = buildReportJobsHref({
    reportKey: 'fraud-evidence-pack-bulk',
    format: 'zip',
  });

  return (
    <PageLayout
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
            <Link to={jobsHref}>Export jobs</Link>
          </MetaLinksBand>
        </DirectoryStack>
      }
      description="ZIP of signed fraud evidence packs per campaign."
      title="Fraud evidence pack bulk"
    >
      <div className="grid gap-4">
        <StubBanner
          title="Export-only report"
          message="Bulk delivery runs through async export jobs. Create a job with report key fraud-evidence-pack-bulk and format zip."
        />
        <p className="m-0 text-sm text-muted-foreground">
          Jobs require customer ID and date range. Completed jobs expose a download URL on the jobs
          page.
        </p>
        <p className="m-0 text-sm">
          <Link className="text-primary hover:underline" to={jobsHref}>
            Open export jobs
          </Link>
        </p>
      </div>
    </PageLayout>
  );
}
