/**
 * Mounts behavioral listeners and unlocks the money page via POST /track/verify.
 */
export function mountSafePageHydrator() {
  const scoreThreshold = 15;
  let score = 0;
  const events: { t: string; ts: number }[] = [];

  const campaignIdFromQuery = (): string => {
    const q = window.location.search;
    if (!q || q.length < 2) {
      return '';
    }
    return new URLSearchParams(q.slice(1)).get('campaign_id') ?? '';
  };

  const trackActivity = (e: Event) => {
    score += 1;
    events.push({ t: e.type, ts: Date.now() });
    if (score >= scoreThreshold) {
      cleanup();
      unlockMoneyPage();
    }
  };

  const cleanup = () => {
    window.removeEventListener('mousemove', trackActivity);
    window.removeEventListener('touchstart', trackActivity);
    window.removeEventListener('scroll', trackActivity);
  };

  const isBot = (): boolean => {
    return navigator.webdriver || !navigator.languages || navigator.languages.length === 0;
  };

  const fingerprint = () => ({
    ua: navigator.userAgent,
    lang: navigator.language,
    languages: [...navigator.languages],
    platform: navigator.platform,
    cores: navigator.hardwareConcurrency ?? 0,
    screen: [window.screen.width, window.screen.height, window.screen.colorDepth],
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    webdriver: Boolean(navigator.webdriver),
  });

  const unlockMoneyPage = async () => {
    if (isBot()) {
      return;
    }
    const campaignId = campaignIdFromQuery();
    if (!campaignId) {
      return;
    }
    const res = await fetch('/track/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        campaign_id: campaignId,
        events,
        fingerprint: fingerprint(),
      }),
    });
    if (!res.ok) {
      return;
    }
    const data = await res.json();
    if (!data?.success || !data.html_content) {
      return;
    }
    document.open();
    document.write(data.html_content);
    document.close();
  };

  window.addEventListener('mousemove', trackActivity);
  window.addEventListener('touchstart', trackActivity);
  window.addEventListener('scroll', trackActivity);
}
