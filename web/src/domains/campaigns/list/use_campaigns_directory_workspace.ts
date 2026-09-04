import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import { fetchCampaignListMetricsBatch } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin } from '@/api/types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import {
  archiveCampaigns,
  bulkPauseOrResumeCampaigns,
} from '@/domains/campaigns/list/campaign_list_bulk_actions';
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
import { resolveCampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import {
  loadCampaignListColumnPrefs,
  mergeCampaignListColumnWidths,
  saveCampaignListColumnPrefs,
  setCampaignListColumnWidth,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import { resetCampaignListWorkspacePrefs } from '@/domains/campaigns/list/campaign_list_workspace_prefs';
import {
  computeCampaignListColumnWidths,
  defaultCampaignListColumnWidths,
} from '@/domains/campaigns/list/campaign_list_column_widths';
import type { CampaignListColumnWidthProbe } from '@/domains/campaigns/list/campaigns_directory_types';
import type { CampaignListFilterTotalsView } from '@/domains/campaigns/list/campaign_list_filter_totals';
import type { CampaignStatsQuery } from '@/api/types';

type UseCampaignsDirectoryWorkspaceArgs = {
  items: Campaign[];
  customerNameById: Record<string, string>;
  ownerEmailById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  columnWidthProbe?: CampaignListColumnWidthProbe;
  filterTotals?: CampaignListFilterTotalsView;
  exportFilterQuery: CampaignListFilterQuery;
  statsQuery: CampaignStatsQuery;
  listScopeKey: string;
  onRefreshList: () => void;
};

export function useCampaignsDirectoryWorkspace({
  items,
  customerNameById,
  ownerEmailById,
  metricsById,
  marginsById,
  columnWidthProbe,
  filterTotals,
  exportFilterQuery,
  statsQuery,
  listScopeKey,
  onRefreshList,
}: UseCampaignsDirectoryWorkspaceArgs) {
  const navigate = useNavigate();
  const [importOpen, setImportOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [cloneOpen, setCloneOpen] = useState(false);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const [resetWorkspaceOpen, setResetWorkspaceOpen] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [exportBusy, setExportBusy] = useState(false);
  const [overviewCampaign, setOverviewCampaign] = useState<CampaignWithMoneyDisplay | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [columnPrefs, setColumnPrefs] = useState<CampaignListColumnPrefs>(() =>
    loadCampaignListColumnPrefs(),
  );

  useEffect(() => {
    setSelectedIds(new Set());
  }, [listScopeKey]);

  const handleColumnPrefsApply = useCallback((prefs: CampaignListColumnPrefs) => {
    setColumnPrefs(prefs);
    saveCampaignListColumnPrefs(prefs);
  }, []);

  const handleResetWorkspaceConfirm = useCallback(() => {
    const prefs = resetCampaignListWorkspacePrefs();
    setColumnPrefs(prefs.columnPrefs);
    setResetWorkspaceOpen(false);
    toast.success('Campaign list view reset');
  }, []);

  const visibleColumns = useMemo(
    () => visibleCampaignListColumns(columnPrefs),
    [columnPrefs],
  );

  const computedColumnWidths = useMemo((): Record<CampaignListColumnId, number> => {
    if (!columnWidthProbe?.items.length) {
      return defaultCampaignListColumnWidths(visibleColumns);
    }
    return computeCampaignListColumnWidths({
      columns: visibleColumns,
      items: columnWidthProbe.items,
      metricsById: columnWidthProbe.metricsById,
      marginsById: columnWidthProbe.marginsById,
      customerNameById,
      ownerEmailById,
      filterTotals,
    });
  }, [columnWidthProbe, customerNameById, filterTotals, ownerEmailById, visibleColumns]);

  const columnWidths = useMemo(
    () => mergeCampaignListColumnWidths(computedColumnWidths, columnPrefs.widthPx, visibleColumns),
    [columnPrefs.widthPx, computedColumnWidths, visibleColumns],
  );

  const handleColumnWidthCommit = useCallback((columnId: CampaignListColumnId, widthPx: number) => {
    setColumnPrefs((current) => {
      const next = setCampaignListColumnWidth(current, columnId, widthPx);
      saveCampaignListColumnPrefs(next);
      return next;
    });
  }, []);

  const selectedCampaignId = useMemo(() => {
    if (selectedIds.size !== 1) {
      return undefined;
    }
    return [...selectedIds][0];
  }, [selectedIds]);

  const selectedCampaign = useMemo(
    () => items.find((item) => item.id === selectedCampaignId),
    [items, selectedCampaignId],
  );

  const summary = useMemo(
    () => resolveCampaignListSummary(items, selectedIds, metricsById, marginsById, filterTotals),
    [filterTotals, items, marginsById, metricsById, selectedIds],
  );

  const selectedIdsList = useMemo(() => [...selectedIds], [selectedIds]);

  const runBulkAction = useCallback(
    async (label: string, action: () => Promise<{ succeeded: string[]; failed: { id: string; error: string }[] }>) => {
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
          onRefreshList();
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err));
      } finally {
        setBulkBusy(false);
        setArchiveOpen(false);
      }
    },
    [onRefreshList, selectedIdsList],
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
      const fromPage = items.filter((item) => selected.has(item.id));
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
  }, [exportFilterQuery, items, selectedIdsList]);

  const onExportCsv = useCallback(() => {
    setExportBusy(true);
    const exportColumns = exportableCampaignListColumns(visibleColumns);
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
          exportColumns,
          batch.metricsById,
          batch.marginsById,
          customerNameById,
          ownerEmailById,
        );
        exportCampaignRowsCsv(exportColumns, exportRows);
        toast.success(
          formatCampaignListExportToast(
            dataset.items.length,
            dataset.matchedTotal,
            dataset.truncated,
            'CSV',
          ),
        );
      })
      .catch((err: unknown) => {
        toast.error(err instanceof Error ? err.message : String(err));
      })
      .finally(() => setExportBusy(false));
  }, [
    customerNameById,
    ownerEmailById,
    resolveExportCampaigns,
    statsQuery,
    visibleColumns,
  ]);

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
            'JSON',
          ),
        );
      })
      .catch((err: unknown) => {
        toast.error(err instanceof Error ? err.message : String(err));
      })
      .finally(() => setExportBusy(false));
  }, [resolveExportCampaigns]);

  const onReportClick = useCallback(() => {
    if (selectedIds.size !== 1) {
      toast.error('Select exactly one campaign for report');
      return;
    }
    if (selectedCampaignId) {
      navigate(`/dashboards/campaign/${selectedCampaignId}`);
    }
  }, [navigate, selectedCampaignId, selectedIds.size]);

  return {
    archiveOpen,
    bulkBusy,
    cloneOpen,
    columnPrefs,
    columnWidths,
    exportBusy,
    handleColumnPrefsApply,
    handleColumnWidthCommit,
    handleResetWorkspaceConfirm,
    importOpen,
    onArchiveSelected,
    onExportBundles,
    onExportCsv,
    onPauseSelected,
    onReportClick,
    onResumeSelected,
    overviewCampaign,
    resetWorkspaceOpen,
    selectedCampaign,
    selectedCampaignId,
    selectedIds,
    setArchiveOpen,
    setCloneOpen,
    setImportOpen,
    setOverviewCampaign,
    setResetWorkspaceOpen,
    setSelectedIds,
    setWizardOpen,
    summary,
    wizardOpen,
  };
}
