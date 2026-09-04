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
import {
  campaignListBodyToolsGutterClass,
  campaignListCellContentClass,
  campaignListCellToolsClass,
  campaignListHeaderCellClass,
  campaignListNumClass,
  campaignListSelectCellClass,
  campaignListTdClass,
  campaignListTdNameClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { CopyableText } from '@/shell/copyable_text';
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
  statsCacheRevision: string;
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
  statsCacheRevision,
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
            <td key={columnId} className={cn(campaignListTdClass, 'px-4 text-center')}>
              <div className={campaignListSelectCellClass}>
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
            <td key={columnId} className={cn(campaignListTdClass, campaignListCellToolsClass, campaignListNumClass, 'text-muted-foreground')}>
              <div className={campaignListHeaderCellClass}>
                <div className={campaignListCellContentClass}>
                  <CopyableText label="Campaign ID" mono title={campaign.id} value={vm.displayId} />
                </div>
                <div aria-hidden className={campaignListBodyToolsGutterClass} />
              </div>
            </td>
          );
        }

        if (columnId === 'name') {
          return (
            <td key={columnId} className={cn(campaignListTdClass, campaignListTdNameClass, campaignListCellToolsClass)}>
              <div className="flex min-h-[34px] max-w-full flex-nowrap items-center gap-1">
                <CampaignCountryBadges
                  className="shrink-0"
                  compact
                  countries={vm.countries}
                  max={2}
                />
                <span
                  className="min-w-0 flex-1 select-text whitespace-nowrap font-medium text-foreground"
                  title={vm.rawName}
                  onClick={(event) => event.stopPropagation()}
                >
                  {vm.rawName}
                </span>
                <CampaignListTableRowMenu
                  className="absolute right-1 top-1/2 -translate-y-1/2"
                  campaign={campaign}
                  onOpenOverview={onCampaignOverview}
                />
              </div>
            </td>
          );
        }

        if (!isCampaignListMiddleColumnId(columnId)) {
          return <td key={columnId} />;
        }

        return (
          <td key={columnId} className={cn(campaignListTdClass, campaignListCellToolsClass, isNum && campaignListNumClass)}>
            <div className={campaignListHeaderCellClass}>
              <div className={campaignListCellContentClass}>
                <CampaignListTableMiddleCell
                campaign={campaign}
                columnId={columnId as CampaignListMiddleColumnId}
                listMetrics={metrics}
                marginBreach={margin?.margin_breach === true}
                statsCacheRevision={statsCacheRevision}
                statsQuery={statsQuery}
                vm={vm}
                onOpenOverview={onCampaignOverview}
              />
              </div>
              <div aria-hidden className={campaignListBodyToolsGutterClass} />
            </div>
          </td>
        );
      })}
    </tr>
  );
}
