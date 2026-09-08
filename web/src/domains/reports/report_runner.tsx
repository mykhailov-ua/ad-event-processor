import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import {
  FilterApplyButton,
  PrimaryActionButton,
  SecondaryActionButton,
} from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_SUMMARY_CLASS,
} from '@/shell/filter_panel';
import { ReportMapTable } from '@/shell/report_map_table';
import { StubBanner } from '@/shell/stub_banner';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { CampaignStats, DataFreshness, FraudEvidencePack, ReportMapRow } from '@/api/types';
import { buildReportJobsHref } from '@/lib/report_paths';
import { reportsLicenseStub, reportsPanelError } from '@/domains/reports/reports_panel_error';
import { deriveColumns } from '@/lib/report_table';
import { displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand } from '@/shell/ui_bands';

export type ReportRunnerProps = {
  reportKey: string;
  title: string;
  description?: string;
  mode: 'table' | 'evidence' | 'export-only' | 'campaign-stats' | 'unsupported';
  rows: ReportMapRow[];
  columns: string[];
  freshness?: DataFreshness;
  nextCursor?: string;
  evidencePack?: FraudEvidencePack;
  campaignStats?: CampaignStats;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftClickId: string;
  limit: number;
  offset: number;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated?: boolean;
  licenseFeatureKey?: string;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftClickIdChange: (value: string) => void;
  onApplyFilters: () => void;
  onPageChange: (nextOffset: number) => void;
  showTelegramExport?: boolean;
  exportingTelegram?: boolean;
  telegramExportError?: Error;
  telegramExportMessage?: string;
  onExportTelegram?: () => void;
};

export function ReportRunner({
  reportKey,
  title,
  description,
  mode,
  rows,
  columns,
  freshness,
  nextCursor,
  evidencePack,
  campaignStats,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftClickId,
  limit,
  offset,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  licenseGated = false,
  licenseFeatureKey,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onDraftClickIdChange,
  onApplyFilters,
  onPageChange,
  showTelegramExport = false,
  exportingTelegram = false,
  telegramExportError,
  telegramExportMessage,
  onExportTelegram,
}: ReportRunnerProps) {
  const timelineColumns = useMemo(
    () => deriveColumns((evidencePack?.timeline ?? []) as ReportMapRow[]),
    [evidencePack?.timeline]
  );
  const fraudEventColumns = useMemo(
    () => deriveColumns((evidencePack?.fraud_events ?? []) as ReportMapRow[]),
    [evidencePack?.fraud_events]
  );

  if (licenseGated) {
    return (
      <PageChrome title={title}>
        <div className="grid gap-3">
          <Link className="text-sm text-muted-foreground hover:underline" to="/reports">
            Back to catalog
          </Link>
          {reportsLicenseStub(licenseFeatureKey)}
        </div>
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title={title}>
        {reportsPanelError(error, `Could not load report ${reportKey}`)}
      </PageChrome>
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = Boolean(nextCursor) || rows.length >= limit;

  return (
    <PageChrome
      title={title}
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">stale CH lag {freshness.ch_lag_seconds ?? '?'}s</Badge>
        ) : freshness ? (
          <Badge variant="outline">{freshness.consistency ?? 'fresh'}</Badge>
        ) : undefined
      }
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Back to catalog</Link>
            {description ? <span>{description}</span> : null}
            {freshness?.as_of ? (
              <span>As of {displayTimestamp(freshness.as_of, freshness.as_of_display)}</span>
            ) : null}
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm
              layout="auto-fill"
              onSubmit={(event) => {
                event.preventDefault();
                onApplyFilters();
              }}
            >
              <FilterField htmlFor="report-customer-id" label="Customer ID">
                <Input
                  id="report-customer-id"
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <DatetimePicker
                id="report-from"
                label="From"
                value={draftFrom}
                onChange={onDraftFromChange}
              />
              <DatetimePicker id="report-to" label="To" value={draftTo} onChange={onDraftToChange} />
              <FilterField htmlFor="report-campaign-id" label="Campaign ID">
                <Input
                  id="report-campaign-id"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              {mode === 'evidence' ? (
                <FilterField htmlFor="report-click-id" label="Click ID">
                  <Input
                    id="report-click-id"
                    value={draftClickId}
                    onChange={(event) => onDraftClickIdChange(event.target.value)}
                  />
                </FilterField>
              ) : null}
              <FilterApplyButton disabled={fetching}>Run report</FilterApplyButton>
              {showTelegramExport && onExportTelegram ? (
                <SecondaryActionButton
                  disabled={exportingTelegram}
                  loading={exportingTelegram}
                  onClick={onExportTelegram}
                  type="button"
                  variant="secondary"
                >
                  Export Telegram bundle
                </SecondaryActionButton>
              ) : null}
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      footer={
        mode === 'table' ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            variant="outline"
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : undefined
      }
    >
      {telegramExportMessage ? (
        <p className="text-sm text-muted-foreground" role="status">
          {telegramExportMessage}
        </p>
      ) : null}
      {telegramExportError ? (
        <ErrorBlock title="Telegram export failed" message={telegramExportError.message} />
      ) : null}

      {mode === 'export-only' ? (
        <StubBanner
          title="Export-only report"
          message="Bulk delivery runs through async export jobs. Open the jobs page with this report key pre-filled."
        />
      ) : null}

      {mode === 'export-only' ? (
        <p className="m-0 text-sm">
          <Link
            className="text-primary hover:underline"
            to={buildReportJobsHref({
              reportKey,
              customerId: draftCustomerId,
              from: draftFrom,
              to: draftTo,
            })}
          >
            Open export jobs
          </Link>
        </p>
      ) : null}

      {mode === 'unsupported' ? (
        <StubBanner
          title="Unsupported report key"
          message={`No runner mapping for report key "${reportKey}".`}
        />
      ) : null}

      {mode === 'campaign-stats' && campaignStats ? (
        <JsonPayloadView payload={campaignStats} />
      ) : null}

      {mode === 'campaign-stats' && !campaignStats && !fetching && !error ? (
        <EmptyState
          title="Campaign ID required"
          description="Enter a campaign ID and run the report."
        />
      ) : null}

      {mode === 'evidence' && evidencePack ? (
        <div className="grid gap-4">
          <CardSummary evidencePack={evidencePack} />
          {evidencePack.timeline && evidencePack.timeline.length > 0 ? (
            <ReportMapTable
              caption="Timeline"
              columns={timelineColumns}
              rows={evidencePack.timeline as ReportMapRow[]}
            />
          ) : null}
          {evidencePack.fraud_events && evidencePack.fraud_events.length > 0 ? (
            <ReportMapTable
              caption="Fraud events"
              columns={fraudEventColumns}
              rows={evidencePack.fraud_events as ReportMapRow[]}
            />
          ) : null}
        </div>
      ) : null}

      {mode === 'table' ? (
        rows.length === 0 ? (
          <EmptyState title="No rows" description="Adjust filters and run the report again." />
        ) : (
          <ReportMapTable
            columns={columns}
            revalidating={listRevalidating}
            rowKeyPrefix={reportKey}
            rows={rows}
          />
        )
      ) : null}

      {error && hasSnapshot ? reportsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}

function CardSummary({ evidencePack }: { evidencePack: FraudEvidencePack }) {
  return (
    <div className={FILTER_PANEL_SUMMARY_CLASS}>
      <div className="flex flex-wrap gap-4">
        <span>Click: {evidencePack.click_id}</span>
        <span>Customer: {evidencePack.customer_id}</span>
        {evidencePack.campaign_id ? <span>Campaign: {evidencePack.campaign_id}</span> : null}
      </div>
      <p className="m-0 text-muted-foreground">
        Generated {displayTimestamp(evidencePack.generated_at, evidencePack.generated_at_display)} |
        digest {evidencePack.digest_sha256}
      </p>
    </div>
  );
}
