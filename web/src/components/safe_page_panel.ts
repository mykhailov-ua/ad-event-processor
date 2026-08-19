export function mountSafePageHydrator() {
  const scoreThreshold = 15;
  let score = 0;
  const events: { t: string; ts: number; x?: number; y?: number }[] = [];

  const campaignIdFromQuery = (): string => {
    const q = window.location.search;
    if (!q || q.length < 2) {
      return '';
    }
    return new URLSearchParams(q.slice(1)).get('campaign_id') ?? '';
  };

  const trackActivity = (e: Event) => {
    score += 1;
    const entry: { t: string; ts: number; x?: number; y?: number } = {
      t: e.type,
      ts: Date.now(),
    };
    if (e.type === 'mousemove') {
      const me = e as MouseEvent;
      entry.x = me.clientX;
      entry.y = me.clientY;
    }
    events.push(entry);
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

  const bytesToHex = (buf: ArrayBuffer): string => {
    const view = new Uint8Array(buf);
    let out = '';
    for (let i = 0; i < view.length; i += 1) {
      out += view[i].toString(16).padStart(2, '0');
    }
    return out;
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
      const gl = (canvas.getContext('webgl') ??
        canvas.getContext('experimental-webgl')) as WebGLRenderingContext | null;
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

  const collectCanvasHash = async (): Promise<string> => {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = 200;
      canvas.height = 50;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        return '';
      }
      ctx.textBaseline = 'top';
      ctx.font = '14px Arial';
      ctx.fillStyle = '#f60';
      ctx.fillRect(0, 0, 200, 50);
      ctx.fillStyle = '#069';
      ctx.fillText('ad-event-processor', 2, 15);
      ctx.font = '18px serif';
      ctx.fillStyle = 'rgba(102, 204, 0, 0.7)';
      ctx.fillText('safe', 4, 25);
      const data = canvas.toDataURL();
      const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(data));
      return bytesToHex(digest);
    } catch {
      return '';
    }
  };

  const collectAudioHash = async (): Promise<string> => {
    try {
      const ctx = new AudioContext();
      const osc = ctx.createOscillator();
      osc.type = 'triangle';
      const filt = ctx.createBiquadFilter();
      filt.type = 'lowpass';
      const analyser = ctx.createAnalyser();
      osc.connect(filt);
      filt.connect(analyser);
      analyser.connect(ctx.destination);
      osc.start(0);
      await new Promise<void>((resolve) => window.setTimeout(resolve, 50));
      const bins = new Uint8Array(analyser.frequencyBinCount);
      analyser.getByteFrequencyData(bins);
      osc.stop();
      await ctx.close();
      const slice = bins.subarray(0, 64);
      const digest = await crypto.subtle.digest('SHA-256', slice);
      return bytesToHex(digest);
    } catch {
      return '';
    }
  };

  const collectNotificationStates = async (): Promise<{ permission: string; query: string }> => {
    const permission = typeof Notification !== 'undefined' ? Notification.permission : 'default';
    try {
      const perm = await navigator.permissions.query({ name: 'notifications' as PermissionName });
      return { permission, query: perm.state };
    } catch {
      return { permission, query: permission };
    }
  };

  const fingerprint = async () => {
    const notif = await collectNotificationStates();
    return {
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
      canvas_hash: await collectCanvasHash(),
      audio_hash: await collectAudioHash(),
      notification_permission: notif.permission,
      notification_query: notif.query,
    };
  };

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
