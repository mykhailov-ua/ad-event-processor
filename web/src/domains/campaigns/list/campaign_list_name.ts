const CAMPAIGN_NAME_PART_SEP = /\s*[\u00b7\u2022|]\s*/;

export type CampaignListNameParts = {
  title: string;
  meta: string[];
};

export function parseCampaignListName(raw: string): CampaignListNameParts {
  const trimmed = raw.trim();
  if (!trimmed) {
    return { title: '', meta: [] };
  }

  const parts = trimmed
    .split(CAMPAIGN_NAME_PART_SEP)
    .map((part) => part.trim())
    .filter(Boolean);

  if (parts.length <= 1) {
    return { title: trimmed, meta: [] };
  }

  return {
    title: parts[0] ?? trimmed,
    meta: parts.slice(1),
  };
}
