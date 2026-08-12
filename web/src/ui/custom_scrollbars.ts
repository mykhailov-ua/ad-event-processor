/**
 * Overlay square scrollbars for every scrollport.
 * Native Firefox thumbs stay rounded — hide them and track scroll live via fixed rails.
 *
 * Performance-optimized: targeted selector scanning and 0-thrash event updates.
 */

const SIZE = 10;
const MIN_THUMB = 24;

type Host = {
  el: HTMLElement;
  viewport: boolean;
  yRail: HTMLElement;
  yThumb: HTMLElement;
  xRail: HTMLElement;
  xThumb: HTMLElement;
  dragging: boolean;
};

type Box = {
  left: number;
  top: number;
  width: number;
  height: number;
};

type RailStyleBox = Box & { styleZoom: number };

type HostMetrics = {
  left: number;
  top: number;
  width: number;
  height: number;
  scrollTop: number;
  scrollLeft: number;
  scrollHeight: number;
  scrollWidth: number;
  clientHeight: number;
  clientWidth: number;
  styleZoom: number;
};

const hosts = new Map<HTMLElement | Window, Host>();
let rafId = 0;
let destroyed = false;
let mo: MutationObserver | null = null;

type CssZoomStyle = CSSStyleDeclaration & { zoom?: string };

/**
 * Read CSS zoom from an element style declaration.
 */
function cssZoom(style: CSSStyleDeclaration): number {
  const raw = Number.parseFloat((style as CssZoomStyle).zoom ?? '');
  return Number.isFinite(raw) && raw > 0 ? raw : 1;
}

/**
 * Rails live on unzoomed document.body so fixed coords match visual viewport pixels.
 */
function railMount(): HTMLElement {
  return document.body || document.documentElement;
}

/**
 * Effective CSS zoom on the app root (html * #root).
 */
function pageZoom(): number {
  const htmlZ = cssZoom(getComputedStyle(document.documentElement));
  const rootEl = document.getElementById('root');
  const rootZ = rootEl ? cssZoom(getComputedStyle(rootEl)) : 1;
  return htmlZ * rootZ;
}

/**
 * Visual viewport box for an element (post-zoom pixels).
 * Some engines report pre-zoom layout boxes under CSS zoom — normalize to visual.
 */
function visualBox(el: HTMLElement): Box {
  const rect = el.getBoundingClientRect();
  const z = pageZoom();
  if (z === 1 || el.offsetWidth <= 0) {
    return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
  }
  const ratio = rect.width / el.offsetWidth;
  if (Math.abs(ratio - 1) <= Math.abs(ratio - z)) {
    return {
      left: rect.left * z,
      top: rect.top * z,
      width: rect.width * z,
      height: rect.height * z,
    };
  }
  return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
}

/**
 * Convert visual box into style pixels for fixed rails on document.body.
 * body is not under #root zoom; html zoom still scales fixed styles.
 */
function toRailStyle(box: Box): RailStyleBox {
  const htmlZ = cssZoom(getComputedStyle(document.documentElement));
  return {
    left: box.left / htmlZ,
    top: box.top / htmlZ,
    width: box.width / htmlZ,
    height: box.height / htmlZ,
    styleZoom: htmlZ,
  };
}

/**
 * Create a scrollbar rail for the given axis.
 */
function makeRail(axis: 'y' | 'x'): { rail: HTMLElement; thumb: HTMLElement } {
  const rail = document.createElement('div');
  rail.className = `csb-rail csb-rail--${axis}`;
  rail.setAttribute('aria-hidden', 'true');
  const thumb = document.createElement('div');
  thumb.className = 'csb-thumb';
  rail.appendChild(thumb);
  railMount().appendChild(rail);
  return { rail, thumb };
}

/**
 * Return true when the element is one of our overlay rails/thumbs.
 */
function isOurUi(el: Element | null | undefined): boolean {
  return el instanceof HTMLElement
    && (el.classList.contains('csb-rail') || el.classList.contains('csb-thumb'));
}

/**
 * Fast targeted check for scroll containers (avoid attaching every overflow:auto node).
 */
function isScrollContainer(el: HTMLElement): boolean {
  if (!(el instanceof HTMLElement)) return false;
  if (el === document.documentElement || el === document.body) return false;
  if (el.classList.contains('no-scrollbar') || isOurUi(el)) return false;
  return el.classList.contains('main')
    || el.classList.contains('table-wrapper')
    || el.classList.contains('modal__body')
    || el.classList.contains('drawer__body')
    || el.classList.contains('login-box__eula')
    || el.hasAttribute('data-scrollable');
}

/**
 * Compute layout and scroll metrics for a host.
 */
function metrics(host: Host): HostMetrics {
  if (host.viewport) {
    const de = document.documentElement;
    const box = toRailStyle({
      left: 0,
      top: 0,
      width: window.innerWidth,
      height: window.innerHeight,
    });
    return {
      left: box.left,
      top: box.top,
      width: box.width,
      height: box.height,
      scrollTop: window.scrollY || de.scrollTop,
      scrollLeft: window.scrollX || de.scrollLeft,
      scrollHeight: Math.max(de.scrollHeight, document.body?.scrollHeight ?? 0),
      scrollWidth: Math.max(de.scrollWidth, document.body?.scrollWidth ?? 0),
      clientHeight: window.innerHeight,
      clientWidth: window.innerWidth,
      styleZoom: box.styleZoom,
    };
  }
  const el = host.el;
  const box = toRailStyle(visualBox(el));
  return {
    left: box.left,
    top: box.top,
    width: box.width,
    height: box.height,
    scrollTop: el.scrollTop,
    scrollLeft: el.scrollLeft,
    scrollHeight: el.scrollHeight,
    scrollWidth: el.scrollWidth,
    clientHeight: el.clientHeight,
    clientWidth: el.clientWidth,
    styleZoom: box.styleZoom,
  };
}

/**
 * Position rails/thumbs for one host.
 */
function updateHost(host: Host): void {
  if (destroyed) return;
  if (!host.viewport && !host.el.isConnected) {
    detach(host);
    return;
  }

  const m = metrics(host);
  if (m.width < 8 || m.height < 8) {
    host.yRail.hidden = true;
    host.xRail.hidden = true;
    return;
  }

  const needY = m.scrollHeight > m.clientHeight + 1;
  const needX = m.scrollWidth > m.clientWidth + 1;
  host.yRail.hidden = !needY;
  host.xRail.hidden = !needX;
  if (!needY && !needX) return;

  if (needY) {
    const track = Math.max(m.height - (needX ? SIZE : 0), 0);
    const thumbH = Math.min(track, Math.max(MIN_THUMB, Math.round((m.clientHeight / m.scrollHeight) * track)));
    const maxTop = Math.max(track - thumbH, 0);
    const maxScroll = Math.max(m.scrollHeight - m.clientHeight, 1);
    const top = maxTop === 0 ? 0 : (m.scrollTop / maxScroll) * maxTop;
    const rail = host.yRail;
    rail.style.left = `${Math.round(m.left + m.width - SIZE)}px`;
    rail.style.top = `${Math.round(m.top)}px`;
    rail.style.width = `${SIZE}px`;
    rail.style.height = `${Math.round(track)}px`;
    host.yThumb.style.height = `${Math.round(thumbH)}px`;
    host.yThumb.style.transform = `translate3d(0, ${Math.round(top)}px, 0)`;
  }

  if (needX) {
    const track = Math.max(m.width - (needY ? SIZE : 0), 0);
    const thumbW = Math.min(track, Math.max(MIN_THUMB, Math.round((m.clientWidth / m.scrollWidth) * track)));
    const maxLeft = Math.max(track - thumbW, 0);
    const maxScroll = Math.max(m.scrollWidth - m.clientWidth, 1);
    const left = maxLeft === 0 ? 0 : (m.scrollLeft / maxScroll) * maxLeft;
    const rail = host.xRail;
    rail.style.left = `${Math.round(m.left)}px`;
    rail.style.top = `${Math.round(m.top + m.height - SIZE)}px`;
    rail.style.width = `${Math.round(track)}px`;
    rail.style.height = `${SIZE}px`;
    host.xThumb.style.width = `${Math.round(thumbW)}px`;
    host.xThumb.style.transform = `translate3d(${Math.round(left)}px, 0, 0)`;
  }
}

function updateAll(): void {
  for (const host of hosts.values()) updateHost(host);
}

function scheduleUpdate(): void {
  if (destroyed || rafId) return;
  rafId = requestAnimationFrame(() => {
    rafId = 0;
    updateAll();
  });
}

/**
 * Detach rails for a host and drop it from the map.
 */
function detach(host: Host): void {
  host.yRail.remove();
  host.xRail.remove();
  if (host.viewport) hosts.delete(window);
  else hosts.delete(host.el);
}

/**
 * Attach rails to a scroll element (or the document viewport).
 */
function attach(el: HTMLElement, opts: { viewport?: boolean } = {}): Host {
  const viewport = Boolean(opts.viewport);
  const key: HTMLElement | Window = viewport ? window : el;
  const existing = hosts.get(key);
  if (existing) {
    const mount = railMount();
    if (existing.yRail.parentElement !== mount) {
      mount.appendChild(existing.yRail);
      mount.appendChild(existing.xRail);
    }
    updateHost(existing);
    return existing;
  }

  const y = makeRail('y');
  const x = makeRail('x');
  const host: Host = {
    el,
    viewport,
    yRail: y.rail,
    yThumb: y.thumb,
    xRail: x.rail,
    xThumb: x.thumb,
    dragging: false,
  };
  bindDrag(host, 'y');
  bindDrag(host, 'x');
  hosts.set(key, host);
  updateHost(host);
  return host;
}

/**
 * Bind thumb/rail pointer interactions for one axis.
 */
function bindDrag(host: Host, axis: 'y' | 'x'): void {
  const thumb = axis === 'y' ? host.yThumb : host.xThumb;
  const rail = axis === 'y' ? host.yRail : host.xRail;

  function applyScroll(next: number): void {
    if (host.viewport) {
      if (axis === 'y') window.scrollTo(window.scrollX, next);
      else window.scrollTo(next, window.scrollY);
      return;
    }
    if (axis === 'y') host.el.scrollTop = next;
    else host.el.scrollLeft = next;
  }

  thumb.addEventListener('pointerdown', (e: PointerEvent) => {
    e.preventDefault();
    e.stopPropagation();
    host.dragging = true;
    thumb.setPointerCapture?.(e.pointerId);
    const startPos = axis === 'y' ? e.clientY : e.clientX;
    const m0 = metrics(host);
    const startScroll = axis === 'y' ? m0.scrollTop : m0.scrollLeft;
    const styleZoom = m0.styleZoom || 1;
    const track = axis === 'y'
      ? Math.max(m0.height - (!host.xRail.hidden ? SIZE : 0), 0)
      : Math.max(m0.width - (!host.yRail.hidden ? SIZE : 0), 0);
    const thumbSize = axis === 'y' ? thumb.offsetHeight : thumb.offsetWidth;
    const maxThumb = Math.max(track - thumbSize, 1);
    const maxScroll = Math.max(
      (axis === 'y' ? m0.scrollHeight - m0.clientHeight : m0.scrollWidth - m0.clientWidth),
      1,
    );

    const onMove = (ev: PointerEvent): void => {
      const deltaVisual = (axis === 'y' ? ev.clientY : ev.clientX) - startPos;
      const delta = deltaVisual / styleZoom;
      applyScroll(startScroll + (delta / maxThumb) * maxScroll);
      updateHost(host);
    };
    const onUp = (): void => {
      host.dragging = false;
      window.removeEventListener('pointermove', onMove);
      window.removeEventListener('pointerup', onUp);
      updateHost(host);
    };
    window.addEventListener('pointermove', onMove);
    window.addEventListener('pointerup', onUp);
  });

  rail.addEventListener('pointerdown', (e: PointerEvent) => {
    if (e.target === thumb) return;
    e.preventDefault();
    const rect = rail.getBoundingClientRect();
    const m0 = metrics(host);
    const track = axis === 'y'
      ? Math.max(m0.height - (!host.xRail.hidden ? SIZE : 0), 0)
      : Math.max(m0.width - (!host.yRail.hidden ? SIZE : 0), 0);
    const thumbSize = axis === 'y' ? thumb.offsetHeight : thumb.offsetWidth;
    const railVisual = axis === 'y' ? rect.height : rect.width;
    const clickVisual = axis === 'y' ? e.clientY - rect.top : e.clientX - rect.left;
    const click = railVisual > 0 ? (clickVisual / railVisual) * track : 0;
    const ratio = Math.min(Math.max((click - thumbSize / 2) / Math.max(track - thumbSize, 1), 0), 1);
    const maxScroll = Math.max(
      (axis === 'y' ? m0.scrollHeight - m0.clientHeight : m0.scrollWidth - m0.clientWidth),
      0,
    );
    applyScroll(ratio * maxScroll);
    updateHost(host);
  });
}

function scan(): void {
  if (destroyed || !document.body) return;
  // SPA shell scrolls inside .main / sidebar — skip document viewport rail.
  const candidates = document.querySelectorAll(
    '.main, .table-wrapper, .modal__body, .drawer__body, .login-box__eula, [data-scrollable]',
  );
  for (let i = 0; i < candidates.length; i++) {
    const el = candidates[i];
    if (el instanceof HTMLElement && isScrollContainer(el)) attach(el);
  }
  for (const [key, host] of [...hosts.entries()]) {
    if (host.viewport) {
      detach(host);
      continue;
    }
    if (!host.el.isConnected) detach(host);
    else if (key instanceof HTMLElement && !isScrollContainer(key) && !host.dragging) {
      detach(host);
    }
  }
  updateAll();
}

/**
 * Install live custom square scrollbars.
 */
export function installCustomScrollbars(): { destroy: () => void } {
  destroyed = false;
  let scanScheduled = false;
  const scheduleScan = (): void => {
    if (destroyed || scanScheduled) return;
    scanScheduled = true;
    requestAnimationFrame(() => {
      scanScheduled = false;
      scan();
    });
  };

  scan();

  const onScroll = (e: Event): void => {
    const t = e.target;
    if (t === document || t === document.documentElement || t === document.body) {
      const host = hosts.get(window);
      if (host) updateHost(host);
      else scheduleUpdate();
      return;
    }
    if (!(t instanceof HTMLElement) || isOurUi(t)) return;
    if (isScrollContainer(t) || hosts.has(t)) {
      const host = attach(t);
      updateHost(host);
    }
    scheduleUpdate();
  };
  document.addEventListener('scroll', onScroll, { capture: true, passive: true });

  const onResize = (): void => {
    scheduleScan();
    scheduleUpdate();
  };
  window.addEventListener('resize', onResize, { passive: true });

  // childList only — watching `style` re-enters forever when rails update position.
  mo = new MutationObserver((records) => {
    for (const rec of records) {
      if (rec.type !== 'childList') continue;
      const nodes = [...rec.addedNodes, ...rec.removedNodes];
      if (nodes.every((n) => n instanceof Element && isOurUi(n))) continue;
      scheduleScan();
      return;
    }
  });
  mo.observe(document.documentElement, {
    childList: true,
    subtree: true,
  });

  const ro = new ResizeObserver(() => scheduleUpdate());
  ro.observe(document.documentElement);
  if (document.body) ro.observe(document.body);

  return {
    destroy() {
      destroyed = true;
      mo?.disconnect();
      mo = null;
      ro.disconnect();
      document.removeEventListener('scroll', onScroll, { capture: true });
      window.removeEventListener('resize', onResize);
      if (rafId) cancelAnimationFrame(rafId);
      rafId = 0;
      for (const host of [...hosts.values()]) detach(host);
      hosts.clear();
    },
  };
}
