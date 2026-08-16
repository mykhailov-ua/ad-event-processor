import { useEffect, useState } from 'react';
import { AlertBanner } from './alert_banner.js';

type RateLimitedDetail = {
  retryAfterSec?: number;
};

/**
 * Offline + rate-limit banners for the shell main content slot.
 */
export function ShellStatusBanners() {
  const [offline, setOffline] = useState(() => !navigator.onLine);
  const [rateLimitDismissed, setRateLimitDismissed] = useState(false);
  const [rateLimitMessage, setRateLimitMessage] = useState<string | null>(null);

  useEffect(() => {
    const onOnlineStatus = () => setOffline(!navigator.onLine);
    const onNetworkError = () => {
      if (!navigator.onLine) setOffline(true);
    };
    const onRateLimited = (e: Event) => {
      const detail = (e as CustomEvent<RateLimitedDetail>).detail;
      const retryAfterSec = detail?.retryAfterSec;
      const message = retryAfterSec
        ? `Rate limited — retry in ${retryAfterSec}s`
        : 'Rate limited — slow down and retry shortly';
      setRateLimitDismissed(false);
      setRateLimitMessage(message);
      if (retryAfterSec && retryAfterSec > 0) {
        window.setTimeout(() => {
          setRateLimitMessage(null);
        }, retryAfterSec * 1000);
      }
    };

    onOnlineStatus();
    window.addEventListener('online', onOnlineStatus);
    window.addEventListener('offline', onOnlineStatus);
    window.addEventListener('admin:network-error', onNetworkError);
    window.addEventListener('admin:rate-limited', onRateLimited);

    return () => {
      window.removeEventListener('online', onOnlineStatus);
      window.removeEventListener('offline', onOnlineStatus);
      window.removeEventListener('admin:network-error', onNetworkError);
      window.removeEventListener('admin:rate-limited', onRateLimited);
    };
  }, []);

  return (
    <>
      {offline ? (
        <AlertBanner
          variant="error"
          message="You are offline. Changes will not sync until the connection returns."
          dismissKey="shell.offline"
          onDismiss={() => setOffline(false)}
        />
      ) : null}
      {rateLimitMessage && !rateLimitDismissed ? (
        <AlertBanner
          variant="warning"
          message={rateLimitMessage}
          dismissKey="shell.rate_limit"
          onDismiss={() => {
            setRateLimitDismissed(true);
            setRateLimitMessage(null);
          }}
        />
      ) : null}
    </>
  );
}
