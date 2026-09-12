import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import {
  EVIDENCE_FRAUD_COLUMNS,
  EVIDENCE_TIMELINE_COLUMNS,
  type EvidencePackColumn,
  type EvidencePackReportConfig,
} from '@/domains/reports/evidence_pack_report_meta';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import {
  buildReportColumnOverviewFields,
  directoryOperateRowsIndexed,
  directoryRecordMapIndexed,
  reportRowLabelFromColumn,
} from '@/domains/reports/report_select_overview';
import { FilterApplyButton } from '@/shell/action_buttons';
import { DirectorySelectOverviewTable } from '@/shell/directory_select_overview_table';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_SUMMARY_CLASS,
} from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { FraudEvidencePack } from '@/api/types';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { displayCount, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';

export type EvidencePackReportDirectoryProps = {
  config: EvidencePackReportConfig;
  evidencePack?: FraudEvidencePack;
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftClickId: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  clickReady: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftClickIdChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
};

function EvidenceSelectOverviewTable<Row extends Record<string, unknown>>({
  caption,
  columns,
  fetching,
  listRevalidating,
  rows,
  rowKeyPrefix,
}: {
  caption: string;
  columns: EvidencePackColumn<Row>[];
  fetching: boolean;
  listRevalidating: boolean;
  rows: Row[];
  rowKeyPrefix: string;
}) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const nameColumn = columns[0];
  const nameColumnLabel = nameColumn?.label ?? 'Event';

  const recordById = useMemo(
    () => directoryRecordMapIndexed(rows, (_row, index) => `${rowKeyPrefix}-${index}`),
    [rowKeyPrefix, rows]
  );
  const operateRows = useMemo(
    () =>
      directoryOperateRowsIndexed(
        rows,
        (_row, index) => `${rowKeyPrefix}-${index}`,
        (row) => reportRowLabelFromColumn(row, nameColumn, caption)
      ),
    [caption, nameColumn, rowKeyPrefix, rows]
  );
  const buildOverviewFields = useCallback(
    (row: Row) =>
      buildReportColumnOverviewFields(row, columns, {
        skipColumnIds: nameColumn ? [nameColumn.id] : undefined,
      }),
    [columns, nameColumn]
  );

  if (rows.length === 0) {
    return null;
  }

  return (
    <section className="grid gap-2">
      <h3 className={adminTypography.caption}>{caption}</h3>
      <TableHost className="w-full">
        <DirectorySelectOverviewTable
          buildOverviewFields={buildOverviewFields}
          disabled={fetching}
          nameColumnLabel={nameColumnLabel}
          overviewTitle={(row) => String(reportRowLabelFromColumn(row, nameColumn, caption))}
          recordById={recordById}
          revalidating={listRevalidating}
          rows={operateRows}
          selectedId={selectedId}
          onSelectedIdChange={setSelectedId}
        />
      </TableHost>
    </section>
  );
}

export function EvidencePackReportDirectory({
  config,
  evidencePack,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftClickId,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  clickReady,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onDraftClickIdChange,
  onApplyFilters,
}: EvidencePackReportDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${config.title}`} message={error.message} />;
  }

  const timeline = evidencePack?.timeline ?? [];
  const fraudEvents = evidencePack?.fraud_events ?? [];
  const signals = evidencePack?.signals;

  return (
    <PageLayout
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor={`${config.key}-customer`} label="Customer ID" wide>
                <Input
                  id={`${config.key}-customer`}
                  value={draftCustomerId}
                  onChange={(event) => onDraftCustomerIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor={`${config.key}-click`} label="Click ID" wide>
                <Input
                  id={`${config.key}-click`}
                  value={draftClickId}
                  onChange={(event) => onDraftClickIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor={`${config.key}-from`} label="From">
                <DatetimePicker
                  id={`${config.key}-from`}
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor={`${config.key}-to`} label="To">
                <DatetimePicker
                  id={`${config.key}-to`}
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterField htmlFor={`${config.key}-campaign`} label="Campaign ID">
                <Input
                  id={`${config.key}-campaign`}
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description={config.description}
      title={config.title}
    >
      {!clickReady ? (
        <EmptyState title="Click ID required" description="Enter click ID and apply filters." />
      ) : !evidencePack ? (
        <EmptyState title="No evidence pack" description="No signed pack for this click window." />
      ) : (
        <TableHost className="grid w-full gap-4">
          <div className={FILTER_PANEL_SUMMARY_CLASS}>
            <div className={`flex flex-wrap gap-4 ${adminTypography.body}`}>
              <span>Click: {evidencePack.click_id}</span>
              <span>Customer: {evidencePack.customer_id}</span>
              {evidencePack.campaign_id ? <span>Campaign: {evidencePack.campaign_id}</span> : null}
            </div>
            <p className={`m-0 ${adminTypography.bodyMuted}`}>
              Generated{' '}
              {displayTimestamp(evidencePack.generated_at, evidencePack.generated_at_display)} |
              digest {evidencePack.digest_sha256}
            </p>
          </div>

          {signals ? (
            <ReportKpiGrid
              items={[
                {
                  label: 'Max fraud score',
                  value: displayCount(signals.max_fraud_score) || '-',
                },
                {
                  label: 'Max layer desync',
                  value: displayCount(signals.max_layer_desync_count) || '-',
                },
                {
                  label: 'Silent reject events',
                  value: displayCount(signals.silent_reject_events) || '-',
                },
                {
                  label: 'Fraud reasons',
                  value: signals.fraud_reasons?.length ? signals.fraud_reasons.join(', ') : '-',
                },
              ]}
            />
          ) : null}

          <EvidenceSelectOverviewTable
            caption="Timeline"
            columns={EVIDENCE_TIMELINE_COLUMNS}
            fetching={fetching}
            listRevalidating={listRevalidating}
            rowKeyPrefix="timeline"
            rows={timeline}
          />

          <EvidenceSelectOverviewTable
            caption="Fraud events"
            columns={EVIDENCE_FRAUD_COLUMNS}
            fetching={fetching}
            listRevalidating={listRevalidating}
            rowKeyPrefix="fraud"
            rows={fraudEvents}
          />
        </TableHost>
      )}
    </PageLayout>
  );
}
