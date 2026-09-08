import { Link, Navigate, useLocation, useParams } from 'react-router-dom';

import { useResource } from '@/api/use_resource';
import { fetchReportCatalogCached } from '@/lib/report_catalog_cache';
import { resolveReportCatalogKey, typedReportRedirectPath } from '@/lib/report_paths';
import { PageLayout } from '@/shell/page_layout';
import { StubBanner } from '@/shell/stub_banner';
import { DirectoryStack, MetaLinksBand } from '@/shell/ui_bands';

type ReportRunnerPageProps = {
  reportKey?: string;
};

export function ReportRunnerPage({ reportKey: reportKeyProp }: ReportRunnerPageProps) {
  const { key } = useParams<{ key?: string }>();
  const location = useLocation();
  const rawKey = reportKeyProp ?? key ?? '';
  const resolvedKey = resolveReportCatalogKey(decodeURIComponent(rawKey));
  const redirectPath = typedReportRedirectPath(resolvedKey);

  const { data: catalog } = useResource((signal) => fetchReportCatalogCached(signal), []);
  const catalogRow = catalog?.rows?.find((row) => row.key === resolvedKey);

  if (redirectPath) {
    return <Navigate replace to={`${redirectPath}${location.search}`} />;
  }

  return (
    <PageLayout
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
        </DirectoryStack>
      }
      description={catalogRow?.description}
      title={catalogRow?.title ?? (resolvedKey || 'Report')}
    >
      <StubBanner
        title="Unknown report"
        message={
          resolvedKey
            ? `No admin page is registered for report key "${resolvedKey}".`
            : 'Missing report key in URL.'
        }
      />
    </PageLayout>
  );
}
