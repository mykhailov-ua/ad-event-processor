import { useMemo } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, CampaignStatsQuery } from '@/api/types';
import { CampaignCountryBadges } from '@/domains/campaigns/list/campaign_country_badges';
import {
  isCampaignListMiddleColumnId,
  isCampaignListNumericColumn,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import { buildCampaignRowVm } from '@/domains/campaigns/list/campaign_list_row_vm';
import { CampaignListTableMiddleCell } from '@/domains/campaigns/list/campaign_list_table_middle_cell';
import { CampaignListTableRowMenu } from '@/domains/campaigns/list/campaign_list_table_row_menu';
import { cn } from '@/lib/utils';

export type CampaignListTableBodyRowProps = {
  campaign: Campaign;
  columns: CampaignListColumnId[];
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
  metrics?: CampaignListMetrics;
  margin?: CampaignMargin;
  selected: boolean;
  fetching: boolean;
  onToggleSelected: (campaignId: string, checked: boolean) => void;
  onCampaignOverview?: (campaign: Campaign) => void;
  statsQuery?: CampaignStatsQuery;
};

export function isCampaignListInteractiveRowTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  return Boolean(
    target.closest('a, button, input, label, [role="checkbox"], [data-col-resize], [data-col-grip]'),
  );
}

export function CampaignListTableBodyRow({
  campaign,
  columns,
  customerNameById,
  ownerEmailById,
  metrics,
  margin,
  selected,
  fetching,
  onToggleSelected,
  onCampaignOverview,
  statsQuery,
}: CampaignListTableBodyRowProps) {
  const vm = useMemo(
    () =>
      buildCampaignRowVm(
        campaign,
        metrics,
        margin,
        customerNameById,
        ownerEmailById,
        selected,
      ),
    [campaign, customerNameById, margin, metrics, ownerEmailById, selected],
  );

  return (
    <tr
      className={cn(vm.rowClass, 'cursor-pointer')}
      onClick={(event) => {
        if (fetching || isCampaignListInteractiveRowTarget(event.target)) {
          return;
        }
        onToggleSelected(campaign.id, !selected);
      }}
    >
      {columns.map((columnId) => {
        const isNum = isCampaignListNumericColumn(columnId);

        if (columnId === 'select') {
          return (
            <td key={columnId} className="w-7 px-1 text-center">
              <div className="admin-table-cell--select">
                <Checkbox
                  aria-label={`Select ${campaign.name}`}
                  checked={selected}
                  disabled={fetching}
                  onCheckedChange={(checked) => onToggleSelected(campaign.id, checked === true)}
                  onClick={(event) => event.stopPropagation()}
                />
              </div>
            </td>
          );
        }

        if (columnId === 'id') {
          return (
            <td
              key={columnId}
              className="num font-mono text-xs text-zinc-500 dark:text-zinc-400"
              title={campaign.id}
            >
              <span className="tabular-nums font-mono text-xs text-zinc-600 dark:text-zinc-400">
                {vm.displayId}
              </span>
            </td>
          );
        }

        if (columnId === 'name') {
          return (
            <td key={columnId} className="whitespace-nowrap">
              <div className="flex min-w-0 items-center gap-1.5">
                <div className="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden">
                  <CampaignCountryBadges
                    className="shrink-0 text-xs text-zinc-500"
                    compact
                    countries={vm.countries}
                    max={2}
                  />
                  <button
                    className="min-w-0 truncate text-left font-medium hover:underline"
                    title={vm.rawName}
                    type="button"
                    onClick={(event) => {
                      event.stopPropagation();
                      onCampaignOverview?.(campaign);
                    }}
                  >
                    {vm.rawName}
                  </button>
                </div>
                <CampaignListTableRowMenu campaign={campaign} onOpenOverview={onCampaignOverview} />
              </div>
            </td>
          );
        }

        if (!isCampaignListMiddleColumnId(columnId)) {
          return <td key={columnId} />;
        }

        return (
          <td key={columnId} className={isNum ? 'num' : undefined}>
            <CampaignListTableMiddleCell
              campaign={campaign}
              columnId={columnId as CampaignListMiddleColumnId}
              marginBreach={margin?.margin_breach === true}
              statsQuery={statsQuery}
              vm={vm}
              onOpenOverview={onCampaignOverview}
            />
          </td>
        );
      })}
    </tr>
  );
}
