import { memo, useMemo } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, CampaignStatsQuery } from '@/api/types';
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
  campaignListNameRowCellClass,
  campaignListNameRowMenuSlotClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
  campaignListNumClass,
  campaignListSelectCellClass,
  campaignListTdClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { CopyButton } from '@/shell/copy_button';
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

export const CampaignListTableBodyRow = memo(function CampaignListTableBodyRow({
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
    () => buildCampaignRowVm(campaign, metrics, margin, customerNameById, ownerEmailById, selected),
    [campaign, customerNameById, margin, metrics, ownerEmailById, selected]
  );

  return (
    <tr className={vm.rowClass}>
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
            <td
              key={columnId}
              className={cn(
                campaignListTdClass,
                campaignListCellToolsClass,
                campaignListNumClass,
                'text-muted-foreground'
              )}
            >
              <div className={campaignListHeaderCellClass}>
                <div className={campaignListCellContentClass}>
                  <span
                    className="select-text whitespace-nowrap font-mono text-xs tabular-nums"
                    title={campaign.id}
                  >
                    {vm.displayId}
                  </span>
                </div>
                <CopyButton
                  flashOnCopy
                  label="Campaign ID"
                  showToast={false}
                  value={vm.displayId}
                />
              </div>
            </td>
          );
        }

        if (columnId === 'name') {
          return (
            <td key={columnId} className={cn(campaignListTdClass, campaignListCellToolsClass)}>
              <div className={campaignListNameRowCellClass}>
                <div className={campaignListNameRowTextClass}>
                  <span className={campaignListNameTextClass} title={vm.rawName}>
                    {vm.rawName}
                  </span>
                </div>
                <div className={campaignListNameRowMenuSlotClass}>
                  <CampaignListTableRowMenu
                    campaign={campaign}
                    onOpenOverview={onCampaignOverview}
                  />
                </div>
              </div>
            </td>
          );
        }

        if (!isCampaignListMiddleColumnId(columnId)) {
          return <td key={columnId} />;
        }

        return (
          <td
            key={columnId}
            className={cn(
              campaignListTdClass,
              campaignListCellToolsClass,
              isNum && campaignListNumClass
            )}
          >
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
});
