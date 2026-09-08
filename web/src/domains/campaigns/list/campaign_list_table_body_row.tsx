import { memo, useMemo } from 'react';

import { Checkbox } from '@/components/ui/checkbox';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, CampaignStatsQuery } from '@/api/types';
import {
  isCampaignListMiddleColumnId,
  isCampaignListNumericColumn,
  isCampaignListPinnedColumn,
  type CampaignListColumnId,
  type CampaignListMiddleColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListPinnedCellClassName,
  campaignListPinnedColumnStyle,
} from '@/domains/campaigns/list/campaign_list_pinned_columns';
import {
  buildCampaignRowVm,
  type CampaignRowVm,
} from '@/domains/campaigns/list/campaign_list_row_vm';
import { campaignListRowDataAttributes } from '@/domains/campaigns/list/campaign_list_row_tone';
import { CampaignListRowAlertBadge } from '@/domains/campaigns/list/campaign_list_row_alert_badge';
import { CampaignListTableMiddleCell } from '@/domains/campaigns/list/campaign_list_table_middle_cell';
import { CampaignListTableRowMenu } from '@/domains/campaigns/list/campaign_list_table_row_menu';
import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListCopyRowClass,
  campaignListCopyTextClass,
  campaignListCopyToolsSlotClass,
  campaignListIdCellInnerClass,
  campaignListNameRowCellClass,
  campaignListNameRowMenuSlotClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
  campaignListSelectCellClass,
  campaignListStatusCellInnerClass,
  campaignListTdClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { CopyButton } from '@/shell/copy_button';
import { cn } from '@/lib/utils';

export type CampaignListTableBodyRowProps = {
  campaign: Campaign;
  columns: CampaignListColumnId[];
  columnWidths: Record<CampaignListColumnId, number>;
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
  metrics?: CampaignListMetrics;
  margin?: CampaignMargin;
  selected: boolean;
  fetching: boolean;
  onToggleSelected: (campaignId: string, checked: boolean) => void;
  onCampaignOverview?: (campaign: Campaign) => void;
  rowVmCache?: ReadonlyMap<string, CampaignRowVm>;
  statsCacheRevision: string;
  statsQuery?: CampaignStatsQuery;
};

export const CampaignListTableBodyRow = memo(function CampaignListTableBodyRow({
  campaign,
  columns,
  columnWidths,
  customerNameById,
  ownerEmailById,
  metrics,
  margin,
  selected,
  fetching,
  onToggleSelected,
  onCampaignOverview,
  rowVmCache,
  statsCacheRevision,
  statsQuery,
}: CampaignListTableBodyRowProps) {
  const vm = useMemo(() => {
    const cached = rowVmCache?.get(campaign.id);
    if (cached) {
      return cached;
    }
    return buildCampaignRowVm(campaign, metrics, margin, customerNameById, ownerEmailById);
  }, [campaign, customerNameById, margin, metrics, ownerEmailById, rowVmCache]);

  const rowDataAttributes = campaignListRowDataAttributes(selected, vm.rowAccent);

  function pinnedCellProps(columnId: CampaignListColumnId) {
    if (!isCampaignListPinnedColumn(columnId)) {
      return {};
    }
    return {
      'data-col-pin': columnId,
      className: campaignListPinnedCellClassName(columnId, columns, 'body'),
      style: campaignListPinnedColumnStyle(columnId, columns, columnWidths),
    };
  }

  return (
    <tr {...rowDataAttributes}>
      {columns.map((columnId) => {
        const isNum = isCampaignListNumericColumn(columnId);
        const pin = pinnedCellProps(columnId);

        if (columnId === 'select') {
          return (
            <td
              key={columnId}
              {...pin}
              className={cn(campaignListTdClass, 'p-0', pin.className)}
              style={pin.style}
            >
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
              {...pin}
              className={cn(campaignListTdClass, 'p-0', pin.className)}
              style={pin.style}
            >
              <div className={campaignListIdCellInnerClass}>
                <div className={campaignListCopyToolsSlotClass}>
                  <CopyButton
                    flashOnCopy
                    label="Campaign ID"
                    showToast={false}
                    value={vm.displayId}
                  />
                </div>
                <span className={campaignListCopyTextClass} title={campaign.id}>
                  {vm.displayId}
                </span>
              </div>
            </td>
          );
        }

        if (columnId === 'name') {
          return (
            <td
              key={columnId}
              {...pin}
              className={cn(campaignListTdClass, 'overflow-visible', pin.className)}
              style={pin.style}
            >
              <div className={campaignListNameRowCellClass}>
                <div
                  className={cn(campaignListNameRowTextClass, 'flex min-w-0 items-center gap-1.5')}
                >
                  {vm.rowAlert !== 'none' ? (
                    <CampaignListRowAlertBadge
                      alert={vm.rowAlert}
                      budgetUsedPct={vm.budgetPct}
                      marginBreach={
                        vm.rowAlert === 'critical' &&
                        (margin?.margin_breach === true || campaign.margin_breach === true)
                      }
                      status={campaign.status}
                    />
                  ) : null}
                  {onCampaignOverview ? (
                    <button
                      className={cn(campaignListNameTextClass, 'text-left hover:underline')}
                      title={vm.rawName}
                      type="button"
                      onClick={() => onCampaignOverview(campaign)}
                    >
                      {vm.rawName}
                    </button>
                  ) : (
                    <span className={campaignListNameTextClass} title={vm.rawName}>
                      {vm.rawName}
                    </span>
                  )}
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

        if (columnId === 'status') {
          return (
            <td
              key={columnId}
              className={cn(campaignListTdClass, 'p-0', vm.statusCellClass)}
              title={vm.statusLabel}
            >
              <div className={campaignListStatusCellInnerClass}>
                <span className="whitespace-nowrap text-[13px] font-medium leading-[18px]">
                  {vm.statusLabel}
                </span>
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
            className={cn(campaignListTdClass, columnId === 'countries' && 'overflow-visible')}
          >
            <div className={isNum ? campaignListCellContentNumClass : campaignListCellContentClass}>
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
          </td>
        );
      })}
    </tr>
  );
});
