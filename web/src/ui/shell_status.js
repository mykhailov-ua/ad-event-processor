import { renderAlertBanner } from './alert_banner.js';

/**
 * Install offline and rate-limit banners at the top of main content.
 *
 * @param {HTMLElement} bannerSlot
 * @returns {{ destroy: () => void, prependTo: (slot: HTMLElement, nodes: HTMLElement[]) => void }}
 */
export function installShellStatus(bannerSlot) {
  let offlineNode = null;
  let rateLimitNode = null;
  let rateLimitTimer = null;

  function renderSlot() {
    const nodes = [];
    if (offlineNode) nodes.push(offlineNode);
    if (rateLimitNode) nodes.push(rateLimitNode);
    bannerSlot.replaceChildren(...nodes);
  }

  function setOffline(offline) {
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

  function onRateLimited(e) {
    const retryAfterSec = e.detail?.retryAfterSec;
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

  function onOnlineStatus() {
    setOffline(!navigator.onLine);
  }

  function onNetworkError() {
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
      const statusNodes = [];
      if (offlineNode) statusNodes.push(offlineNode);
      if (rateLimitNode) statusNodes.push(rateLimitNode);
      slot.replaceChildren(...statusNodes, ...nodes);
    },
  };
}
