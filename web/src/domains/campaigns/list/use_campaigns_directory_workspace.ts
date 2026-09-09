// workspace owner: campaigns directory selection, export, bulk actions.
// Fetch fan-out lives in use_campaigns_page_list.ts; this hook consumes list snapshots only.
// listScopeKey change clears row selection and popover stats cache (statsRevision in parent).
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { fetchCampaignListMetricsBatch } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { useCampaignImportPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_import_panel_workspace';
import { useCampaignWizardPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_wizard_panel_workspace';
import {
  archiveCampaigns,
  bulkPauseOrResumeCampaigns,
} from '@/domains/campaigns/list/campaign_list_bulk_actions';
import {
  defaultCampaignListColumnPrefs,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  exportCampaignBundles,
  exportCampaignRowsCsv,
  formatCampaignListExportToast,
  listAllCampaignsForFilter,
  type CampaignListExportDataset,
} from '@/domains/campaigns/list/campaign_list_export';
import {
  buildCampaignListExportRows,
  exportableCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_export_rows';
import type { CampaignListFilterQuery } from '@/domains/campaigns/list/campaigns_list_query';
import { clearCampaignStatsCache } from '@/domains/campaigns/list/campaign_list_stats_cache';
import type { CampaignStatsQuery } from '@/api/types';
import { invalidateCampaignListResponseCache } from '@/domains/campaigns/list/campaign_list_response_cache';
import { toError } from '@/lib/admin_error';

const DEFAULT_EXPORT_COLUMNS = exportableCampaignListColumns(
  visibleCampaignListColumns(defaultCampaignListColumnPrefs())
);

type UseCampaignsDirectoryWorkspaceArgs = {
  items?: Campaign[];
  customerOptions: CustomerComboboxOption[];
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  exportFilterQuery: CampaignListFilterQuery;
  statsQuery: CampaignStatsQuery;
  listScopeKey: string;
  onRefreshList: () => void;
};

export function useCampaignsDirectoryWorkspace({
  items,
  customerOptions,
  customerNameById,
  ownerEmailById,
  exportFilterQuery,
  statsQuery,
  listScopeKey,
  onRefreshList,
}: UseCampaignsDirectoryWorkspaceArgs) {
  const listItems = items ?? [];
  const [importOpen, setImportOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const refreshListAfterMutation = useCallback(() => {
    invalidateCampaignListResponseCache();
    onRefreshList();
  }, [onRefreshList]);

  const importPanelWorkspace = useCampaignImportPanelWorkspace(importOpen);
  const wizardPanelWorkspace = useCampaignWizardPanelWorkspace({
    enabled: wizardOpen,
    customerOptions,
    onCampaignCreated: refreshListAfterMutation,
  });
  const [cloneOpen, setCloneOpen] = useState(false);
  const [bulkCloneOpen, setBulkCloneOpen] = useState(false);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [exportBusy, setExportBusy] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    setSelectedIds(new Set());
    clearCampaignStatsCache();
  }, [listScopeKey]);

  const selectedCampaignId = useMemo(() => {
    if (selectedIds.size !== 1) {
      return undefined;
    }
    return [...selectedIds][0];
  }, [selectedIds]);

  const selectedCampaign = useMemo(
    () => listItems.find((item) => item.id === selectedCampaignId),
    [listItems, selectedCampaignId]
  );

  const selectedIdsList = useMemo(() => [...selectedIds], [selectedIds]);

  const runBulkAction = useCallback(
    async (
      label: string,
      action: () => Promise<{ succeeded: string[]; failed: { id: string; error: string }[] }>
    ) => {
      if (selectedIdsList.length === 0) {
        return;
      }
      setBulkBusy(true);
      try {
        const result = await action();
        if (result.succeeded.length > 0) {
          toast.success(`${label}: ${result.succeeded.length} campaign(s)`);
        }
        if (result.failed.length > 0) {
          toast.error(`${label} failed for ${result.failed.length} campaign(s)`);
        }
        if (result.succeeded.length === 0 && result.failed.length === 0) {
          toast.error(`${label}: no campaigns updated`);
        }
        if (result.succeeded.length > 0) {
          setSelectedIds(new Set());
          refreshListAfterMutation();
        }
      } catch (err: unknown) {
        toast.error(toError(err).message);
      } finally {
        setBulkBusy(false);
        setArchiveOpen(false);
      }
    },
    [refreshListAfterMutation, selectedIdsList]
  );

  const onPauseSelected = useCallback(() => {
    void runBulkAction('Paused', () => bulkPauseOrResumeCampaigns('pause', selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const onResumeSelected = useCallback(() => {
    void runBulkAction('Resumed', () => bulkPauseOrResumeCampaigns('resume', selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const onArchiveSelected = useCallback(() => {
    void runBulkAction('Archived', () => archiveCampaigns(selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const resolveExportCampaigns = useCallback(async (): Promise<CampaignListExportDataset> => {
    if (selectedIdsList.length > 0) {
      const selected = new Set(selectedIdsList);
      const fromPage = listItems.filter((item) => selected.has(item.id));
      if (fromPage.length === selectedIdsList.length) {
        return {
          items: fromPage,
          matchedTotal: fromPage.length,
          truncated: false,
        };
      }
      const allFiltered = await listAllCampaignsForFilter(exportFilterQuery);
      const selectedRows = allFiltered.items.filter((item) => selected.has(item.id));
      return {
        items: selectedRows,
        matchedTotal: selectedIdsList.length,
        truncated: selectedRows.length < selectedIdsList.length || allFiltered.truncated,
      };
    }
    return listAllCampaignsForFilter(exportFilterQuery);
  }, [exportFilterQuery, listItems, selectedIdsList]);

  const onExportCsv = useCallback(() => {
    setExportBusy(true);
    void resolveExportCampaigns()
      .then(async (dataset) => {
        if (dataset.items.length === 0) {
          toast.error('No campaigns to export');
          return;
        }
        const campaignIds = dataset.items.map((row) => row.id);
        const batch = await fetchCampaignListMetricsBatch(campaignIds, statsQuery);
        const exportRows = buildCampaignListExportRows(
          dataset.items,
          DEFAULT_EXPORT_COLUMNS,
          batch.metricsById,
          batch.marginsById,
          customerNameById,
          ownerEmailById
        );
        exportCampaignRowsCsv(DEFAULT_EXPORT_COLUMNS, exportRows);
        toast.success(
          formatCampaignListExportToast(
            dataset.items.length,
            dataset.matchedTotal,
            dataset.truncated,
            'CSV'
          )
        );
      })
      .catch((err: unknown) => {
        toast.error(toError(err).message);
      })
      .finally(() => setExportBusy(false));
  }, [customerNameById, ownerEmailById, resolveExportCampaigns, statsQuery]);

  const onExportBundles = useCallback(() => {
    setExportBusy(true);
    void resolveExportCampaigns()
      .then(async (dataset) => {
        if (dataset.items.length === 0) {
          toast.error('No campaigns to export');
          return;
        }
        await exportCampaignBundles(dataset.items.map((row) => row.id));
        toast.success(
          formatCampaignListExportToast(
            dataset.items.length,
            dataset.matchedTotal,
            dataset.truncated,
            'JSON'
          )
        );
      })
      .catch((err: unknown) => {
        toast.error(toError(err).message);
      })
      .finally(() => setExportBusy(false));
  }, [resolveExportCampaigns]);

  return {
    archiveOpen,
    bulkBusy,
    bulkCloneOpen,
    cloneOpen,
    exportBusy,
    importOpen,
    importPanelWorkspace,
    onArchiveSelected,
    onExportBundles,
    onExportCsv,
    onPauseSelected,
    onResumeSelected,
    selectedCampaign,
    selectedCampaignId,
    selectedIds,
    selectedIdsList,
    setArchiveOpen,
    setBulkCloneOpen,
    setCloneOpen,
    setImportOpen,
    setSelectedIds,
    setWizardOpen,
    wizardOpen,
    wizardPanelWorkspace,
    refreshListAfterMutation,
  };
}
