import type { DocsSection } from '@/lib/docs_types';

export const TRACKER_DOCS_SECTION: DocsSection = {
  id: 'tracker',
  title: 'Click & conversion tracking',
  summary:
    'How to send traffic to your campaigns, record conversions, and verify that clicks and postbacks reach the tracker.',
  audience: 'client',
  guides: [
    {
      id: 'overview',
      title: 'How tracking works',
      blocks: [
        {
          type: 'paragraph',
          text: 'The tracker is the edge service that accepts incoming clicks, applies your campaign rules (budget, geo, fraud filters), and records conversions. You do not send buyers directly to an offer URL. You send them to your tracker click URL first; the tracker assigns a click id, debits budget when allowed, and redirects to the lander or offer in your flow.',
        },
        {
          type: 'table',
          headers: ['Endpoint', 'Method', 'Use for'],
          rows: [
            ['/click', 'GET', 'Traffic source click URL (banner, push, native, social ad)'],
            ['/track', 'POST (or GET query)', 'Server postback when a lead or sale happens'],
            ['/static/track.js', 'GET', 'Browser pixel on the landing page (zero-redirect conversions)'],
          ],
        },
        {
          type: 'note',
          text: 'Accepted events return HTTP 202 on the hot path. Budget debit and analytics land in Redis and ClickHouse shortly after; Postgres spend syncs on a cold path. A 202 response means the tracker accepted the event, not that every downstream report row is already visible.',
        },
      ],
    },
    {
      id: 'quick-start',
      title: 'Quick start (five steps)',
      blocks: [
        {
          type: 'list',
          items: [
            'Create a campaign in Campaigns (wizard template or manual). Note the campaign UUID.',
            'Open the campaign Integration tab. Apply a traffic template for your network (Meta, Propeller, MGID, and others) or build a click URL manually.',
            'Paste the click URL into the ad network. Required query param: campaign_id={your-campaign-uuid}. Add network macros for subs and click ids (sub1, fbclid, gclid, and so on).',
            'Wire conversions: affiliate panel postback URL pointing at /track, or embed the browser pixel on the lander (see below).',
            'Smoke test: open the Integration tab test link or hit /click?campaign_id=...&click_id=smoke-test&smoke=1, then confirm Ops metrics or campaign reports show the click.',
          ],
        },
        {
          type: 'paragraph',
          text: 'Keitaro / Voluum mental model: Campaign URL = /click. Postback URL = /track. Both are shown on the campaign Integration tab after you apply templates.',
        },
      ],
    },
    {
      id: 'click-urls',
      title: 'Click URLs (/click)',
      blocks: [
        {
          type: 'paragraph',
          text: 'Every click URL must include campaign_id as a UUID. Replace {track_host} with the hostname shown on the Integration tab (often trk.your-domain.com behind nginx on :443).',
        },
        {
          type: 'code',
          code: 'https://{track_host}/click?campaign_id={campaign_uuid}&sub1={zone_id}&sub2={campaign_name}&fbclid={fbclid}',
        },
        {
          type: 'heading',
          text: 'Common query parameters',
        },
        {
          type: 'table',
          headers: ['Parameter', 'Required', 'Purpose'],
          rows: [
            ['campaign_id', 'Yes', 'Campaign UUID from the admin UI'],
            ['click_id', 'No', 'Your correlation id; generated if omitted on some paths'],
            ['sub1 ... sub30', 'No', 'Labels for zones, placements, creatives, affiliate tokens'],
            ['fbclid, gclid, ttclid, msclkid', 'No', 'Ad network click ids for CAPI / offline conversions'],
            ['utm_source, utm_medium, utm_campaign', 'No', 'UTM tags stored and passed through macros'],
            ['cost, cpc, bid', 'No', 'Ingress spend when campaign ingress cost is enabled'],
          ],
        },
        {
          type: 'note',
          text: 'Templates in Integrations map network-specific tokens to these fields. After apply-templates, the Integration tab shows the exact URL for your source.',
        },
      ],
    },
    {
      id: 'conversions-s2s',
      title: 'Server postbacks (/track)',
      blocks: [
        {
          type: 'paragraph',
          text: 'Affiliate networks and your own backend should call /track when a conversion is approved. The click id (or sub that carried it) must match the original click so revenue attributes to the right placement.',
        },
        {
          type: 'heading',
          text: 'JSON POST (recommended for landers and APIs)',
        },
        {
          type: 'code',
          code: `POST https://{track_host}/track
Content-Type: application/json

{
  "campaign_id": "{campaign_uuid}",
  "type": "conversion",
  "click_id": "{click_id_from_landing_url}",
  "sub1": "{optional_sub}",
  "event_id": "{uuid_for_dedup}"
}`,
        },
        {
          type: 'heading',
          text: 'Affiliate panel URL (GET-style)',
        },
        {
          type: 'paragraph',
          text: 'Many CPA networks accept a postback URL with query parameters. Import an affiliate template (Integrations -> Templates) to get the exact mapping for Everad, Leadbit, AdCombo, and others.',
        },
        {
          type: 'code',
          code: 'https://{track_host}/track?campaign_id={campaign_uuid}&sub1={clickid}&payout={payout}&status={status}',
        },
        {
          type: 'table',
          headers: ['status value', 'Typical meaning'],
          rows: [
            ['approved, sale, lead', 'Counts as approved conversion'],
            ['hold', 'Hold lead (funnel metrics)'],
            ['rejected, reject, declined', 'Rejected lead'],
          ],
        },
      ],
    },
    {
      id: 'browser-pixel',
      title: 'Browser pixel (track.js)',
      blocks: [
        {
          type: 'paragraph',
          text: 'For conversions that happen in the browser (form submit, thank-you page), embed the tracker script on the lander. It POSTs to /track with click_id read from the landing page URL query string.',
        },
        {
          type: 'code',
          code: `<script src="https://{track_host}/static/track.js"></script>
<script>
  trackEvent({
    endpoint: 'https://{track_host}/track',
    campaignId: '{campaign_uuid}',
    type: 'conversion',
    clickId: new URLSearchParams(location.search).get('click_id'),
    eventId: crypto.randomUUID()
  });
</script>`,
        },
        {
          type: 'list',
          items: [
            'Set TRACK_CORS_ORIGINS on the tracker to include your lander origin (comma-separated hostnames).',
            'Copy the ready-made snippet from Campaign -> Integration -> Zero-redirect (browser pixel).',
            'Optional: fire type impression on DOMContentLoaded for LP view diagnostics (does not replace conversion postback).',
            'Verify in browser DevTools: POST /track returns 202; payload includes click_id and event_id.',
          ],
        },
        {
          type: 'note',
          text: 'Optional Meta / Google / TikTok browser tags can share the same conversionEventId variable as the tracker snippet when CAPI postbacks are configured. Server CAPI fires separately via the postback worker after settlement.',
        },
      ],
    },
    {
      id: 'macros',
      title: 'Macros on redirect and offer URLs',
      blocks: [
        {
          type: 'paragraph',
          text: 'When the tracker redirects to a lander or offer, it can expand macros in the destination URL. Use the same click id on the offer side that you will send back on /track.',
        },
        {
          type: 'table',
          headers: ['Macro', 'Expands to'],
          rows: [
            ['{click_id} or {{click_id}}', 'Click correlation id'],
            ['{campaign_id} or {{campaign.id}}', 'Campaign UUID'],
            ['{sub1} ... {sub30}', 'Sub values from the click URL'],
            ['{{fbclid}}, {{gclid}}, {{ttclid}}', 'Network click ids when present on ingress'],
          ],
        },
        {
          type: 'paragraph',
          text: 'Outbound postbacks to your CRM or network use a similar token set: {click_id}, {payout}, {tx_id}, {sub1}, {event_type}. Configure mappings under Integrations -> Postbacks.',
        },
      ],
    },
    {
      id: 'verification',
      title: 'Verification checklist',
      blocks: [
        {
          type: 'list',
          items: [
            'Click test: open the smoke URL from Integration; expect 302 to lander or safe page, not 403/410.',
            'Budget: campaign current spend increases after real clicks (Redis debit is immediate; list view may lag seconds).',
            'Conversion test: fire /track with the click_id from the lander URL; check Reports or campaign metrics for conversions in the stats window.',
            'Browser pixel: DevTools Network tab shows POST /track with 202 and JSON body.',
            'CAPI (optional): use postback test dispatch in Integrations; fix warnings before relying on Meta/Google Events Manager.',
          ],
        },
        {
          type: 'note',
          text: 'Silent reject and fraud blocks may return 202 with decoy content while analytics record a silent_reject_event. Check Fraud settings if clicks look accepted but funnel counts differ.',
        },
      ],
    },
  ],
  topics: [
    {
      problem: '403 or empty response on /click',
      symptom: 'Traffic does not reach the lander; network shows low or zero clicks.',
      fix: 'Confirm campaign is ACTIVE, budget > 0, geo matches target_countries, and campaign_id UUID is correct. Check edge XDP or fraud rules in Ops if only some IPs fail.',
    },
    {
      problem: 'Conversions missing in reports',
      symptom: '/track returns 202 but dashboard shows zero conversions.',
      fix: 'Ensure click_id on postback matches the id from the click URL. Check stats date range on the campaign list. Allow a few minutes for ClickHouse rollups; compare with Ops -> Metrics ingest lag.',
    },
    {
      problem: 'Browser pixel blocked by CORS',
      symptom: 'POST /track fails in console with CORS error.',
      fix: 'Add the lander origin to TRACK_CORS_ORIGINS on the tracker service. Redeploy tracker; hard-refresh the lander.',
    },
    {
      problem: 'Duplicate conversions',
      symptom: 'Same lead counted twice in affiliate and tracker.',
      fix: 'Send a stable event_id per conversion attempt. Align affiliate postback idempotency with tracker dedup; avoid firing both S2S and browser pixel for the same event without shared event_id.',
    },
    {
      problem: 'Macros not replaced on redirect',
      symptom: 'Lander URL still shows {click_id} literals.',
      fix: 'Use supported macro syntax in flow destination URLs. Pass click_id on the inbound /click URL or let the tracker generate one. Re-publish the flow after URL edits.',
    },
  ],
};
