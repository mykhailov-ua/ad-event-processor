import { Link } from 'react-router-dom';
import { reportTitle } from '../../models/report.js';
import { PageChrome } from '../system/page_chrome.js';
import { StubBanner } from '../system/stub_banner.js';
import styles from './report_runner.module.css';

export type ReportStubPageProps = {
  reportKey: string;
};

export function ReportStubPage({ reportKey }: ReportStubPageProps) {
  const title = reportTitle(reportKey);
  return (
    <div className={styles.root}>
      <PageChrome title={title} />
      <StubBanner
        title="Report page not wired"
        message={`No live SPA route is registered for report key "${reportKey}".`}
      />
      <Link to="/reports">Back to reports hub</Link>
    </div>
  );
}
