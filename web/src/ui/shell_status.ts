import { renderAlertBanner } from './alert_banner.js';

export type ShellStatusHandle = {
  destroy: () => void;
  prependTo: (slot: HTMLElement, nodes: HTMLElement[]) => void;
};

type RateLimitedDetail = {
  retryAfterSec?: number;
};

/**
 * Install offline and rate-limit banners at the top of main content.
 */
export function installShellStatus(bannerSlot: HTMLElement): ShellStatusHandle {
  let offlineNode: HTMLElement | null = null;
  let rateLimitNode: HTMLElement | null = null;
  let rateLimitTimer: ReturnType<typeof setTimeout> | null = null;

  function renderSlot(): void {
    const nodes: HTMLElement[] = [];
    if (offlineNode) nodes.push(offlineNode);
    if (rateLimitNode) nodes.push(rateLimitNode);
    bannerSlot.replaceChildren(...nodes);
  }

  function setOffline(offline: boolean): void {
    if (!offline) {
      offlineNode = null;
      renderSlot();
      return;
    }
    offlineNode = renderAlertBanner({
      variant: 'error',
      message: 'You are offline. Changes will not sync until the connection returns.',
      dismissKey: 'shell.offline',
      onDismiss: () => {
        offlineNode = null;
        renderSlot();
      },
    });
    renderSlot();
  }

  function onRateLimited(e: Event): void {
    const detail = (e as CustomEvent<RateLimitedDetail>).detail;
    const retryAfterSec = detail?.retryAfterSec;
    const message = retryAfterSec
      ? `Rate limited — retry in ${retryAfterSec}s`
      : 'Rate limited — slow down and retry shortly';
    rateLimitNode = renderAlertBanner({
      variant: 'warning',
      message,
      dismissKey: 'shell.rate_limit',
      onDismiss: () => {
        rateLimitNode = null;
        if (rateLimitTimer) clearTimeout(rateLimitTimer);
        rateLimitTimer = null;
        renderSlot();
      },
    });
    renderSlot();
    if (rateLimitTimer) clearTimeout(rateLimitTimer);
    if (retryAfterSec && retryAfterSec > 0) {
      rateLimitTimer = setTimeout(() => {
        rateLimitNode = null;
        rateLimitTimer = null;
        renderSlot();
      }, retryAfterSec * 1000);
    }
  }

  function onOnlineStatus(): void {
    setOffline(!navigator.onLine);
  }

  function onNetworkError(): void {
    if (!navigator.onLine) setOffline(true);
  }

  onOnlineStatus();
  window.addEventListener('online', onOnlineStatus);
  window.addEventListener('offline', onOnlineStatus);
  window.addEventListener('admin:network-error', onNetworkError);
  window.addEventListener('admin:rate-limited', onRateLimited);

  return {
    destroy() {
      window.removeEventListener('online', onOnlineStatus);
      window.removeEventListener('offline', onOnlineStatus);
      window.removeEventListener('admin:network-error', onNetworkError);
      window.removeEventListener('admin:rate-limited', onRateLimited);
      if (rateLimitTimer) clearTimeout(rateLimitTimer);
    },
    /** Rebuild banners after version or license slots update. */
    prependTo(slot, nodes) {
      const statusNodes: HTMLElement[] = [];
      if (offlineNode) statusNodes.push(offlineNode);
      if (rateLimitNode) statusNodes.push(rateLimitNode);
      slot.replaceChildren(...statusNodes, ...nodes);
    },
  };
}
