import {
  defaultCampaignListColumnPrefs,
  saveCampaignListColumnPrefs,
  type CampaignListColumnPrefs,
} from './campaign_list_columns.ts';
import {
  defaultCampaignListRowDisplayPrefs,
  saveCampaignListRowDisplayPrefs,
  type CampaignListRowDisplayPrefs,
} from './campaign_list_row_display.ts';

export const CAMPAIGN_LIST_WORKSPACE_RESET_ITEMS = [
  'Visible columns and column order',
  'Column widths',
  'Row highlight preference',
] as const;

export type CampaignListWorkspacePrefs = {
  columnPrefs: CampaignListColumnPrefs;
  rowDisplayPrefs: CampaignListRowDisplayPrefs;
};

export function defaultCampaignListWorkspacePrefs(): CampaignListWorkspacePrefs {
  return {
    columnPrefs: defaultCampaignListColumnPrefs(),
    rowDisplayPrefs: defaultCampaignListRowDisplayPrefs(),
  };
}

export function resetCampaignListWorkspacePrefs(): CampaignListWorkspacePrefs {
  const prefs = defaultCampaignListWorkspacePrefs();
  saveCampaignListColumnPrefs(prefs.columnPrefs);
  saveCampaignListRowDisplayPrefs(prefs.rowDisplayPrefs);
  return prefs;
}
