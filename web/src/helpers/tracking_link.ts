
export function defaultClickTemplate(domain: string): string {
  const host = (domain || '').replace(/^https?:\/\  if (!host) {
    return 'https://track.example/click?campaign_id={campaign_id}&sub1={sub1}';
  }
  return `https://${host}/click?campaign_id={campaign_id}&sub1={sub1}&sub2={sub2}&sub3={sub3}&sub4={sub4}&sub5={sub5}&sub6={sub6}&sub7={sub7}&sub8={sub8}&sub9={sub9}&sub10={sub10}`;
}


export function buildTrackingLink(
  template: string,
  campaignId: string,
  subs: Record<string, string> = {}
): string {
  let url = template.replaceAll('{campaign_id}', encodeURIComponent(campaignId));
  for (let i = 1; i <= 30; i += 1) {
    const token = `{sub${i}}`;
    const val = subs[`sub${i}`] ?? '';
    url = url.replaceAll(token, encodeURIComponent(val));
  }
  return url;
}
