// Eight-digit operator id: prefer API display_id; else hash campaign.id (base 31, mod 90M + 10M).
// Must match internal/campaign/display_id.go CampaignDisplayIDFromString.
export function campaignDisplayId(campaign: { display_id?: string; id: string }): string {
  const fromApi = campaign.display_id?.trim();
  if (fromApi && /^\d{8}$/.test(fromApi)) {
    return fromApi;
  }
  let hash = 0;
  for (let index = 0; index < campaign.id.length; index += 1) {
    hash = (hash * 31 + campaign.id.charCodeAt(index)) >>> 0;
  }
  return String(10000000 + (hash % 90000000)).padStart(8, '0');
}
