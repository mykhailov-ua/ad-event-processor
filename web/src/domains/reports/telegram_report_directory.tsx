import { Link } from 'react-router-dom';

import { FilterApplyButton, SecondaryActionButton } from '@/shell/action_buttons';
import { CustomerCombobox } from '@/shell/customer_combobox';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
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
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';
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
  const columnCount = config.columns?.length ?? 4;

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={Math.min(columnCount, 8)} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${config.title}`} message={error.message} />;
  }

  const badge = freshness?.stale ? (
    <Badge variant="secondary">Stale data</Badge>
  ) : freshness?.lag_seconds != null ? (
    <span className="text-xs text-muted-foreground">CH lag {freshness.lag_seconds}s</span>
  ) : null;

  const summary = config.summaryBand?.(extras);
  const showTable = Boolean(config.columns?.length);

  return (
    <PageLayout
      badge={badge}
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
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
        <p className="text-sm text-muted-foreground">{telegramExportMessage}</p>
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
              <DirectoryTable
                className={cn('w-full', directoryTableRevalidatingClass(listRevalidating))}
                horizontalScroll
                nested
              >
                <TableHeader>
                  <TableRow>
                    {config.columns?.map((column) => (
                      <DirectoryTableHead key={column.id} align={column.align}>
                        {column.label}
                      </DirectoryTableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((row, index) => (
                    <TableRow key={config.rowKey?.(row, index) ?? String(index)}>
                      {config.columns?.map((column) => (
                        <TableCell
                          key={column.id}
                          className={column.align === 'end' ? 'text-right' : undefined}
                        >
                          {column.cell(row)}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            </TableHost>
          ) : null}
        </DirectoryStack>
      )}
    </PageLayout>
  );
}
