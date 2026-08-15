import { useEffect, useRef } from 'react';

export type LegacyPanelHostProps = {
  active: boolean;
  mount: (host: HTMLElement) => { destroy?: () => void } | void;
  deps?: unknown[];
};

/**
 * Mount an imperative legacy panel into a DOM host when active.
 */
export function LegacyPanelHost({ active, mount, deps = [] }: LegacyPanelHostProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!active || !ref.current) return undefined;
    const host = ref.current;
    const handle = mount(host);
    return () => {
      handle?.destroy?.();
      host.replaceChildren();
    };
  }, [active, mount, ...deps]);

  if (!active) return null;
  return <div ref={ref} />;
}

export type ImperativeDomHostProps = {
  build: () => HTMLElement | null;
  deps?: unknown[];
};

/**
 * Render imperative DOM from a legacy builder into a React host.
 */
export function ImperativeDomHost({ build, deps = [] }: ImperativeDomHostProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return undefined;
    const host = ref.current;
    const node = build();
    if (node) host.replaceChildren(node);
    else host.replaceChildren();
    return () => host.replaceChildren();
  }, [build, ...deps]);

  return <div ref={ref} />;
}
