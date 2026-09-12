'use strict';
(() => {
  function parseCampaignId() {
    const q = document.location.search.slice(1).split('&');
    for (let i = 0; i < q.length; i += 1) {
      const part = q[i];
      const eq = part.indexOf('=');
      if (eq < 0) {
        continue;
      }
      if (part.slice(0, eq) === 'campaign_id') {
        return decodeURIComponent(part.slice(eq + 1));
      }
    }
    return '';
  }

  function show() {
    document.documentElement.style.visibility = 'visible';
  }

  document.documentElement.style.visibility = 'hidden';
  const boot = globalThis.tagLiteBoot;
  if (typeof boot !== 'function') {
    show();
    return;
  }
  Promise.resolve(boot(parseCampaignId())).then(show).catch(show);
})();
