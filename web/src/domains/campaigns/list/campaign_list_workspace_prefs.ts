import {
  defaultCampaignListColumnPrefs,
  saveCampaignListColumnPrefs,
  type CampaignListColumnPrefs,
} from './campaign_list_columns.ts';

export const CAMPAIGN_LIST_WORKSPACE_RESET_ITEMS = [
  'Visible columns and column order',
  'Column widths',
] as const;

export type CampaignListWorkspacePrefs = {
  columnPrefs: CampaignListColumnPrefs;
};

export function defaultCampaignListWorkspacePrefs(): CampaignListWorkspacePrefs {
  return {
    columnPrefs: defaultCampaignListColumnPrefs(),
  };
}

export function resetCampaignListWorkspacePrefs(): CampaignListWorkspacePrefs {
  const prefs = defaultCampaignListWorkspacePrefs();
  saveCampaignListColumnPrefs(prefs.columnPrefs);
  return prefs;
}
