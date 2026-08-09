/**
 * Default click URL template for a tracking host.
 *
 * @param {string} domain
 * @returns {string}
 */
export function defaultClickTemplate(domain) {
  const host = (domain || '').replace(/^https?:\/\//, '').trim();
  if (!host) {
    return 'https://track.example/click?campaign_id={campaign_id}&sub1={sub1}';
  }
  return `https://${host}/click?campaign_id={campaign_id}&sub1={sub1}&sub2={sub2}&sub3={sub3}&sub4={sub4}&sub5={sub5}`;
}

/**
 * Substitute campaign id and sub placeholders in a click template.
 *
 * @param {string} template
 * @param {string} campaignId
 * @param {Record<string, string>} [subs]
 * @returns {string}
 */
export function buildTrackingLink(template, campaignId, subs = {}) {
  let url = template.replaceAll('{campaign_id}', encodeURIComponent(campaignId));
  for (let i = 1; i <= 5; i += 1) {
    const token = `{sub${i}}`;
    const val = subs[`sub${i}`] ?? '';
    url = url.replaceAll(token, encodeURIComponent(val));
  }
  return url;
}
