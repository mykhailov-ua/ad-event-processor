import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Table, TableBody, TableHeader } from '@/components/ui/table';
import {
  formatReportMoneyUsd,
  formatReportPct,
  type CampaignReportRow,
} from '@/domains/campaigns/report/campaign_report_rows';
import { cn } from '@/lib/utils';

export type CampaignReportTableProps = {
  rows: CampaignReportRow[];
  selectedIds: Set<string>;
  onSelectedIdsChange: (ids: Set<string>) => void;
  sortColumn?: string;
  sortDesc?: boolean;
  onSort?: (column: string) => void;
};

function rowToneClass(row: CampaignReportRow, selected: boolean): string {
  if (selected) {
    return 'admin-row-selected';
  }
  if (row.profitUsd > 0) {
    return 'admin-row-positive';
  }
  if (row.profitUsd < 0) {
    return 'admin-row-negative';
  }
  return '';
}

function profitClass(value: number): string {
  if (value > 0) {
    return 'admin-positive';
  }
  if (value < 0) {
    return 'admin-negative';
  }
  return 'admin-muted';
}

export function CampaignReportTable({
  rows,
  selectedIds,
  onSelectedIdsChange,
  sortColumn = 'clicks',
  sortDesc = true,
  onSort,
}: CampaignReportTableProps) {
  const allSelected = rows.length > 0 && rows.every((row) => selectedIds.has(row.id));

  function toggleAll(checked: boolean) {
    if (!checked) {
      onSelectedIdsChange(new Set());
      return;
    }
    onSelectedIdsChange(new Set(rows.map((row) => row.id)));
  }

  function toggleOne(id: string, checked: boolean) {
    const next = new Set(selectedIds);
    if (checked) {
      next.add(id);
    } else {
      next.delete(id);
    }
    onSelectedIdsChange(next);
  }

  function headerButton(column: string, label: string) {
    const active = sortColumn === column;
    return (
      <Button type="button" variant="ghost" onClick={() => onSort?.(column)}>
        {label}
        {active ? (sortDesc ? ' v' : ' ^') : ''}
      </Button>
    );
  }

  return (
    <div className="admin-table-wrap">
      <Table bare className="admin-table">
        <TableHeader>
          <tr>
            <th>
              <Checkbox
                aria-label="Select all"
                checked={allSelected}
                onCheckedChange={(checked) => toggleAll(checked === true)}
              />
            </th>
            <th>{headerButton('name', 'Name')}</th>
            <th>Marks</th>
            <th className="num">{headerButton('clicks', 'Clicks')}</th>
            <th className="num">{headerButton('lp_ctr', 'LP CTR')}</th>
            <th className="num">{headerButton('cr', 'CR')}</th>
            <th className="num">{headerButton('lp_clicks', 'LP Clicks')}</th>
            <th className="num">{headerButton('leads', 'Leads')}</th>
            <th className="num">{headerButton('epc', 'EPC')}</th>
            <th className="num">{headerButton('cpc', 'CPC')}</th>
            <th className="num">{headerButton('revenue', 'Revenue')}</th>
            <th className="num">{headerButton('cost', 'Cost')}</th>
            <th className="num">{headerButton('profit', 'Profit')}</th>
            <th className="num">{headerButton('roi', 'ROI')}</th>
          </tr>
        </TableHeader>
        <TableBody>
          {rows.map((row) => {
            const selected = selectedIds.has(row.id);
            return (
              <tr key={row.id} className={rowToneClass(row, selected)}>
                <td>
                  <Checkbox
                    aria-label={`Select ${row.name}`}
                    checked={selected}
                    onCheckedChange={(checked) => toggleOne(row.id, checked === true)}
                  />
                </td>
                <td title={row.name}>{row.name}</td>
                <td>
                  <Button type="button" variant="ghost" size="icon">
                    *
                  </Button>
                </td>
                <td className="num">{row.clicks}</td>
                <td className="num admin-muted">{formatReportPct(row.lpCtrPct)}</td>
                <td className="num">{formatReportPct(row.crPct)}</td>
                <td className="num">{row.lpClicks}</td>
                <td className="num">{row.leads}</td>
                <td className="num">{formatReportMoneyUsd(row.epcUsd)}</td>
                <td className="num">{formatReportMoneyUsd(row.cpcUsd)}</td>
                <td className="num">{formatReportMoneyUsd(row.revenueUsd)}</td>
                <td className="num">{formatReportMoneyUsd(row.costUsd)}</td>
                <td className={cn('num', profitClass(row.profitUsd))}>
                  {formatReportMoneyUsd(row.profitUsd)}
                </td>
                <td className={cn('num', profitClass(row.roiPct))}>{formatReportPct(row.roiPct)}</td>
              </tr>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
