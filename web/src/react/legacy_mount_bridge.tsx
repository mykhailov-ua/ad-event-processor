import { useEffect, useRef } from 'react';
import type { RouteContext, ViewHandle, ViewModule } from '../lib/router_types.js';

export type LegacyMountBridgeProps = {
  load: () => Promise<ViewModule>;
  ctx: RouteContext;
};

/**
 * Bridge legacy mount()/destroy() views inside a React route outlet.
 * Phase 0 scaffold only — not wired until Phase 1 shell migration.
 */
export function LegacyMountBridge({ load, ctx }: LegacyMountBridgeProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<ViewHandle | null>(null);

  useEffect(() => {
    let cancelled = false;
    const host = hostRef.current;
    if (!host) return undefined;

    void (async () => {
      const mod = await load();
      if (cancelled || !hostRef.current) return;
      handleRef.current = mod.mount(hostRef.current, ctx) ?? null;
    })();

    return () => {
      cancelled = true;
      handleRef.current?.destroy?.();
      handleRef.current = null;
    };
  }, [load, ctx]);

  return <div ref={hostRef} className="legacy-mount-bridge" />;
}
