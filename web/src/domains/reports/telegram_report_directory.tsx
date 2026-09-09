import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import { FilterApplyButton, SecondaryActionButton } from '@/shell/action_buttons';
import { CustomerCombobox } from '@/shell/customer_combobox';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { TelegramReportFreshness } from '@/api/types';
import type { TelegramReportConfig } from '@/domains/reports/telegram_report_meta';
import {
  buildReportColumnOverviewFields,
  directoryOperateRowsIndexed,
  directoryRecordMapIndexed,
  reportRowLabelFromColumn,
} from '@/domains/reports/report_select_overview';
import { DirectorySelectOverviewTable } from '@/shell/directory_select_overview_table';
import { buildReportJobsHref } from '@/lib/report_paths';
import { InAppLink } from '@/shell/in_app_link';
import { adminTypography } from '@/lib/admin_kit';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

export type TelegramReportDirectoryProps<Row> = {
  config: TelegramReportConfig<Row>;
  rows: Row[];
  freshness?: TelegramReportFreshness;
  extras?: Record<string, unknown>;
  customerOptions: CustomerComboboxOption[];
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  rangeReady: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  enableExport?: boolean;
  exportingTelegram?: boolean;
  telegramExportError?: Error;
  telegramExportMessage?: string;
  onExportTelegram?: () => void;
};

export function TelegramReportDirectory<Row>({
  config,
  rows,
  freshness,
  extras,
  customerOptions,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  rangeReady,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onApplyFilters,
  enableExport = false,
  exportingTelegram = false,
  telegramExportError,
  telegramExportMessage,
  onExportTelegram,
}: TelegramReportDirectoryProps<Row>) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const columns = config.columns ?? [];
  const nameColumn = columns[0];
  const nameColumnLabel = nameColumn?.label ?? 'Dimension';
  const showTable = columns.length > 0;
  const rowKey = config.rowKey ?? ((_row, index) => `telegram-row-${index}`);

  const recordById = useMemo(
    () => directoryRecordMapIndexed(rows, (row, index) => rowKey(row, index) || null),
    [rowKey, rows]
  );
  const operateRows = useMemo(
    () =>
      directoryOperateRowsIndexed(
        rows,
        (row, index) => rowKey(row, index) || null,
        (row) => reportRowLabelFromColumn(row, nameColumn, 'Telegram row')
      ),
    [nameColumn, rowKey, rows]
  );
  const buildOverviewFields = useCallback(
    (row: Row) =>
      buildReportColumnOverviewFields(row, columns, {
        skipColumnIds: nameColumn ? [nameColumn.id] : undefined,
      }),
    [columns, nameColumn]
  );
  const exportHubHref = useMemo(
    () =>
      buildReportJobsHref({
        reportKey: config.key,
        customerId: draftCustomerId.trim() || undefined,
        from: draftFrom || undefined,
        to: draftTo || undefined,
      }),
    [config.key, draftCustomerId, draftFrom, draftTo]
  );
  const overviewFooter = useCallback(
    (_row: Row) =>
      enableExport ? (
        <InAppLink className="text-primary hover:underline" to={exportHubHref}>
          Open Export hub
        </InAppLink>
      ) : null,
    [enableExport, exportHubHref]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${config.title}`} message={error.message} />;
  }

  const badge = freshness?.stale ? (
    <Badge variant="secondary">Stale data</Badge>
  ) : freshness?.lag_seconds != null ? (
    <span className={adminTypography.bodyMuted}>CH lag {freshness.lag_seconds}s</span>
  ) : null;

  const summary = config.summaryBand?.(extras);

  return (
    <PageLayout
      badge={badge}
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
            {enableExport ? <Link to={exportHubHref}>Export hub</Link> : null}
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor={`${config.key}-customer`} label="Customer" wide>
                <CustomerCombobox
                  id={`${config.key}-customer`}
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
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
              {enableExport ? (
                <SecondaryActionButton
                  disabled={exportingTelegram}
                  type="button"
                  onClick={onExportTelegram}
                >
                  {exportingTelegram ? 'Exporting...' : 'Export bundle'}
                </SecondaryActionButton>
              ) : null}
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description={config.description}
      title={config.title}
    >
      {telegramExportMessage ? (
        <p className={adminTypography.bodyMuted}>{telegramExportMessage}</p>
      ) : null}
      {telegramExportError ? (
        <ErrorBlock title="Telegram export failed" message={telegramExportError.message} />
      ) : null}
      {!rangeReady ? (
        <EmptyState
          title="Date range required"
          description="Set from and to timestamps, then apply filters."
        />
      ) : showTable && rows.length === 0 && !summary ? (
        <EmptyState title="No rows" description="No rows for the current filters." />
      ) : (
        <DirectoryStack>
          {summary ?? null}
          {showTable && rows.length > 0 ? (
            <TableHost className="w-full">
              <DirectorySelectOverviewTable
                buildOverviewFields={buildOverviewFields}
                disabled={fetching}
                nameColumnLabel={nameColumnLabel}
                overviewFooter={enableExport ? overviewFooter : undefined}
                overviewTitle={(row) =>
                  String(reportRowLabelFromColumn(row, nameColumn, config.title))
                }
                recordById={recordById}
                revalidating={listRevalidating}
                rows={operateRows}
                selectedId={selectedId}
                onSelectedIdChange={setSelectedId}
              />
            </TableHost>
          ) : null}
        </DirectoryStack>
      )}
    </PageLayout>
  );
}
