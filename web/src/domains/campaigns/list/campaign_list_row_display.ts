export const CAMPAIGN_LIST_ROW_DISPLAY_STORAGE_KEY = 'aed.campaigns.listRowDisplay.v1';

export type CampaignListRowDisplayPrefs = {
  highlightActiveRows: boolean;
};

export function defaultCampaignListRowDisplayPrefs(): CampaignListRowDisplayPrefs {
  return { highlightActiveRows: false };
}

export function loadCampaignListRowDisplayPrefs(): CampaignListRowDisplayPrefs {
  try {
    const raw = window.localStorage.getItem(CAMPAIGN_LIST_ROW_DISPLAY_STORAGE_KEY);
    if (!raw) {
      return defaultCampaignListRowDisplayPrefs();
    }
    const parsed = JSON.parse(raw) as Partial<CampaignListRowDisplayPrefs>;
    return {
      highlightActiveRows: parsed.highlightActiveRows === true,
    };
  } catch {
    return defaultCampaignListRowDisplayPrefs();
  }
}

export function saveCampaignListRowDisplayPrefs(prefs: CampaignListRowDisplayPrefs): void {
  try {
    window.localStorage.setItem(CAMPAIGN_LIST_ROW_DISPLAY_STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // ignore storage errors
  }
}
