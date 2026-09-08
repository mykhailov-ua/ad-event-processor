import { Link } from 'react-router-dom';
import type { ReactNode } from 'react';

import { FilterApplyButton } from '@/shell/action_buttons';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CampaignToggleCohortKPIRow } from '@/api/types';
import {
  type CampaignToggleField,
  useCampaignToggleCohortPageWorkspace,
} from '@/domains/fraud/use_campaign_toggle_cohort_page_workspace';
import { displayCount, displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

function formatRoi(value: number | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}%`;
}

function renderDelta(row: CampaignToggleCohortKPIRow): ReactNode {
  if (!row.delta_label) {
    return '-';
  }
  const tone = row.delta_tone;
  const variant =
    tone === 'up' || tone === 'warn'
      ? 'secondary'
      : tone === 'down' || tone === 'good'
        ? 'default'
        : 'outline';
  return <Badge variant={variant}>{row.delta_label}</Badge>;
}

export function CampaignToggleCohortDirectory() {
  const {
    rows,
    insufficient,
    toggleAt,
    windowHours,
    freshness,
    draftCampaignId,
    draftToggleField,
    draftToggleAt,
    draftWindowHours,
    fetching,
    listRevalidating,
    error,
    hasSnapshot,
    onDraftCampaignIdChange,
    onDraftToggleFieldChange,
    onDraftToggleAtChange,
    onDraftWindowHoursChange,
    onApplyFilters,
  } = useCampaignToggleCohortPageWorkspace();

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={6} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaign toggle cohort" message={error.message} />;
  }

  const campaignRequired = !draftCampaignId.trim();

  return (
    <PageLayout
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">Stale data</Badge>
        ) : freshness?.as_of ? (
          <span className="text-xs text-muted-foreground">
            As of {displayTimestamp(freshness.as_of)}
          </span>
        ) : insufficient ? (
          <Badge variant="secondary">Insufficient data</Badge>
        ) : null
      }
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/fraud">Fraud hub</Link>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor="toggle-cohort-campaign" label="Campaign ID" wide>
                <Input
                  id="toggle-cohort-campaign"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="toggle-cohort-field" label="Toggle field">
                <Select
                  value={draftToggleField}
                  onValueChange={(value) =>
                    onDraftToggleFieldChange(value as CampaignToggleField)
                  }
                >
                  <SelectTrigger id="toggle-cohort-field" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="silent_reject_enabled">silent_reject_enabled</SelectItem>
                    <SelectItem value="accept_lang_geo_enabled">accept_lang_geo_enabled</SelectItem>
                    <SelectItem value="json_serialization_enabled">
                      json_serialization_enabled
                    </SelectItem>
                  </SelectContent>
                </Select>
              </FilterField>
              <FilterField htmlFor="toggle-cohort-at" label="Toggle at (optional)">
                <DatetimePicker
                  id="toggle-cohort-at"
                  label="Toggle at"
                  value={draftToggleAt}
                  onChange={onDraftToggleAtChange}
                />
              </FilterField>
              <FilterField htmlFor="toggle-cohort-window" label="Window hours">
                <Input
                  id="toggle-cohort-window"
                  inputMode="numeric"
                  min={1}
                  max={168}
                  value={draftWindowHours}
                  onChange={(event) => onDraftWindowHoursChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Before and after KPI windows around a campaign fraud toggle change."
      title="Campaign toggle cohort"
    >
      {toggleAt ? (
        <p className="text-sm text-muted-foreground">
          Toggle at {displayTimestamp(toggleAt)}
          {windowHours ? `; window ${windowHours}h` : null}
        </p>
      ) : null}
      {campaignRequired ? (
        <EmptyState
          title="Campaign required"
          description="Enter a campaign ID and toggle field, then apply filters."
        />
      ) : rows.length === 0 ? (
        <EmptyState title="No rows" description="No cohort data for the current filters." />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable
            className={cn('w-full', directoryTableRevalidatingClass(listRevalidating))}
            horizontalScroll
            nested
          >
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Window</DirectoryTableHead>
                <DirectoryTableHead align="end">Impressions</DirectoryTableHead>
                <DirectoryTableHead align="end">Rejects</DirectoryTableHead>
                <DirectoryTableHead align="end">Conversions</DirectoryTableHead>
                <DirectoryTableHead align="end">ROI</DirectoryTableHead>
                <DirectoryTableHead>Reject delta</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.window ?? 'unknown'}>
                  <TableCell className="font-mono text-xs">{row.window ?? '-'}</TableCell>
                  <TableCell className="text-right">
                    {displayCount(row.impressions ?? 0)}
                  </TableCell>
                  <TableCell className="text-right">{displayCount(row.rejects ?? 0)}</TableCell>
                  <TableCell className="text-right">
                    {displayCount(row.conversions ?? 0)}
                  </TableCell>
                  <TableCell className="text-right">{formatRoi(row.roi_pct)}</TableCell>
                  <TableCell>{renderDelta(row)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
    </PageLayout>
  );
}
