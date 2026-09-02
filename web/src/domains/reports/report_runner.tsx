import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import { FilterApplyButton, PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { PaginationPrevNext } from '@/shell/pagination_prev_next';
import { StubBanner } from '@/shell/stub_banner';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { DataFreshness, FraudEvidencePack, ReportMapRow } from '@/api/types';
import { formatMapCell, deriveColumns, reportMapRowKey } from '@/lib/report_table';
import { displayTimestamp } from '@/lib/display';

export type ReportRunnerProps = {
  reportKey: string;
  title: string;
  description?: string;
  mode: 'table' | 'evidence' | 'export-only' | 'unsupported';
  rows: ReportMapRow[];
  columns: string[];
  freshness?: DataFreshness;
  nextCursor?: string;
  evidencePack?: FraudEvidencePack;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftClickId: string;
  limit: number;
  offset: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftClickIdChange: (value: string) => void;
  onApplyFilters: () => void;
  onPageChange: (nextOffset: number) => void;
  showTelegramExport?: boolean;
  exportingTelegram?: boolean;
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
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftClickId,
  limit,
  offset,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onDraftClickIdChange,
  onApplyFilters,
  onPageChange,
  showTelegramExport = false,
  exportingTelegram = false,
  telegramExportMessage,
  onExportTelegram,
}: ReportRunnerProps) {
  const timelineColumns = useMemo(
    () => deriveColumns((evidencePack?.timeline ?? []) as ReportMapRow[]),
    [evidencePack?.timeline],
  );
  const fraudEventColumns = useMemo(
    () => deriveColumns((evidencePack?.fraud_events ?? []) as ReportMapRow[]),
    [evidencePack?.fraud_events],
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load report ${reportKey}`} message={error.message} />;
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
    >
      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        <Link className="hover:underline" to="/reports">
          Back to catalog
        </Link>
        {description ? <span>{description}</span> : null}
        {freshness?.as_of ? (
          <span>As of {displayTimestamp(freshness.as_of, freshness.as_of_display)}</span>
        ) : null}
      </div>

      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApplyFilters();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="report-customer-id">Customer ID</Label>
          <Input
            id="report-customer-id"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <DatetimePicker
          id="report-from"
          label="From"
          value={draftFrom}
          onChange={onDraftFromChange}
        />
        <DatetimePicker id="report-to" label="To" value={draftTo} onChange={onDraftToChange} />
        <div className="grid gap-2">
          <Label htmlFor="report-campaign-id">Campaign ID</Label>
          <Input
            id="report-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        {mode === 'evidence' ? (
          <div className="grid gap-2">
            <Label htmlFor="report-click-id">Click ID</Label>
            <Input
              id="report-click-id"
              value={draftClickId}
              onChange={(event) => onDraftClickIdChange(event.target.value)}
            />
          </div>
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
        {mode === 'table' ? (
          <PaginationPrevNext
            canGoPrev={canGoPrev}
            canGoNext={canGoNext}
            disabled={fetching}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
            onNext={() => onPageChange(offset + limit)}
          />
        ) : null}
      </form>

      {telegramExportMessage ? (
        <p className="text-sm text-muted-foreground" role="status">
          {telegramExportMessage}
        </p>
      ) : null}

      {mode === 'export-only' ? (
        <StubBanner
          title="Export-only report"
          message="This report is available through async export jobs. Use the Reports jobs API or operator CLI for bulk ZIP delivery."
        />
      ) : null}

      {mode === 'unsupported' ? (
        <StubBanner
          title="Unsupported report key"
          message={`No runner mapping for report key "${reportKey}".`}
        />
      ) : null}

      {mode === 'evidence' && evidencePack ? (
        <div className="grid gap-4">
          <CardSummary evidencePack={evidencePack} />
          {evidencePack.timeline && evidencePack.timeline.length > 0 ? (
            <EvidenceTable
              caption="Timeline"
              columns={timelineColumns}
              rows={evidencePack.timeline as ReportMapRow[]}
            />
          ) : null}
          {evidencePack.fraud_events && evidencePack.fraud_events.length > 0 ? (
            <EvidenceTable
              caption="Fraud events"
              columns={fraudEventColumns}
              rows={evidencePack.fraud_events as ReportMapRow[]}
            />
          ) : null}
        </div>
      ) : null}

      {mode === 'table' ? (
        rows.length === 0 ? (
          <EmptyState
            title="No rows"
            description="Adjust filters and run the report again."
          />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  {columns.map((column) => (
                    <TableHead key={column}>{column}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((row, index) => (
                  <TableRow key={reportMapRowKey(row, columns, index, reportKey)}>
                    {columns.map((column) => (
                      <TableCell key={column}>{formatMapCell(row[column])}</TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )
      ) : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}

function CardSummary({ evidencePack }: { evidencePack: FraudEvidencePack }) {
  return (
    <div className="ui-filter-panel gap-2 text-sm">
      <div className="flex flex-wrap gap-4">
        <span>Click: {evidencePack.click_id}</span>
        <span>Customer: {evidencePack.customer_id}</span>
        {evidencePack.campaign_id ? <span>Campaign: {evidencePack.campaign_id}</span> : null}
      </div>
      <div className="text-muted-foreground">
        Generated {displayTimestamp(evidencePack.generated_at, evidencePack.generated_at_display)}{' '}
        | digest {evidencePack.digest_sha256}
      </div>
    </div>
  );
}

function EvidenceTable({
  caption,
  columns,
  rows,
}: {
  caption: string;
  columns: string[];
  rows: ReportMapRow[];
}) {
  return (
    <div className="ui-table-frame">
      <p className="border-b px-4 py-2 text-sm font-medium">{caption}</p>
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((column) => (
              <TableHead key={column}>{column}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
            <TableRow key={reportMapRowKey(row, columns, index, caption)}>
              {columns.map((column) => (
                <TableCell key={column}>{formatMapCell(row[column])}</TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
