import { Button } from '@/components/ui/button';
import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import { PageLayout } from '@/shell/page_layout';
import { CampaignReportTable } from '@/domains/campaigns/report/campaign_report_table';
import {
  buildCampaignReportRows,
  type CampaignReportDimension,
  type CampaignReportRow,
} from '@/domains/campaigns/report/campaign_report_rows';
import type { Flow, FlowPath } from '@/api/types';

const DIMENSION_PILLS: { id: CampaignReportDimension; label: string }[] = [
  { id: 'default', label: 'My presets' },
  { id: 'paths', label: 'Paths' },
  { id: 'offers', label: 'Offers' },
  { id: 'landers', label: 'Landers' },
  { id: 'rules', label: 'Rules' },
  { id: 'tokens', label: 'Tokens' },
  { id: 'connection', label: 'Connection' },
  { id: 'device', label: 'Device' },
  { id: 'country', label: 'Country' },
];

export type CampaignReportDirectoryProps = {
  campaignId: string;
  campaignName?: string;
  payload: Record<string, unknown> | undefined;
  flow?: Flow;
  fetching?: boolean;
  draftQ: string;
  onDraftQChange: (value: string) => void;
  onRefresh: () => void;
};

function sortRows(
  rows: CampaignReportRow[],
  column: string,
  desc: boolean,
): CampaignReportRow[] {
  const direction = desc ? -1 : 1;
  return [...rows].sort((left, right) => {
    const pick = (row: CampaignReportRow): number | string => {
      switch (column) {
        case 'name':
          return row.name;
        case 'clicks':
          return row.clicks;
        case 'profit':
          return row.profitUsd;
        case 'roi':
          return row.roiPct;
        default:
          return row.clicks;
      }
    };
    const leftValue = pick(left);
    const rightValue = pick(right);
    if (typeof leftValue === 'string' && typeof rightValue === 'string') {
      return leftValue.localeCompare(rightValue) * direction;
    }
    return ((leftValue as number) - (rightValue as number)) * direction;
  });
}

export function CampaignReportDirectory({
  campaignId,
  campaignName,
  payload,
  flow,
  fetching = false,
  draftQ,
  onDraftQChange,
  onRefresh,
}: CampaignReportDirectoryProps) {
  const [dimension, setDimension] = useState<CampaignReportDimension>('paths');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [sortColumn, setSortColumn] = useState('clicks');
  const [sortDesc, setSortDesc] = useState(true);

  const flowPaths: FlowPath[] = Array.isArray(flow?.paths) ? flow.paths : [];

  const rows = useMemo(() => {
    const built = buildCampaignReportRows({
      payload,
      dimension,
      flowPaths,
    });
    const filtered = draftQ.trim()
      ? built.filter((row) => row.name.toLowerCase().includes(draftQ.trim().toLowerCase()))
      : built;
    return sortRows(filtered, sortColumn, sortDesc);
  }, [dimension, draftQ, flowPaths, payload, sortColumn, sortDesc]);

  const title = campaignName ?? 'Campaign report';
  const description = campaignName ? campaignId : undefined;

  return (
    <PageLayout
      controlPanel={
        <div className="admin-stack">
          <div className="admin-toolbar-row">
            {DIMENSION_PILLS.map((pill) => (
              <Button
                key={pill.id}
                type="button"
                variant={dimension === pill.id ? 'default' : 'secondary'}
                onClick={() => setDimension(pill.id)}
              >
                {pill.label}
              </Button>
            ))}
          </div>
          <div className="admin-toolbar-row">
            <label className="admin-label">
              Search
              <input
                className="admin-input"
                value={draftQ}
                onChange={(event) => onDraftQChange(event.target.value)}
              />
            </label>
            <Button disabled={fetching} loading={fetching} type="button" onClick={onRefresh}>
              Refresh
            </Button>
            <Button asChild type="button" variant="secondary">
              <Link to={`/campaigns/${campaignId}/edit`}>Edit</Link>
            </Button>
          </div>
        </div>
      }
      description={description}
      title={title}
    >
      {rows.length === 0 ? (
        <p className="admin-muted">No data in this range.</p>
      ) : (
        <CampaignReportTable
          rows={rows}
          selectedIds={selectedIds}
          sortColumn={sortColumn}
          sortDesc={sortDesc}
          onSelectedIdsChange={setSelectedIds}
          onSort={(column) => {
            if (sortColumn === column) {
              setSortDesc((value) => !value);
              return;
            }
            setSortColumn(column);
            setSortDesc(true);
          }}
        />
      )}
    </PageLayout>
  );
}
