import type { ViewHandle } from './router_types.js';

export type BootHandle = {
  destroy: () => void;
};

/**
 * Boot the authenticated admin SPA into root.
 * @deprecated Legacy path — main app uses react/main_mount.js.
 */
export async function bootApp(_root: HTMLElement): Promise<BootHandle | void> {
  // Retained for type/export compatibility; main.tsx mounts React AppShell directly.
}

/**
 * @deprecated Use react/login_boot.tsx via login.tsx entry.
 */
export async function bootLogin(_root: HTMLElement): Promise<void> {
  // Retained for type/export compatibility; login.tsx mounts React directly.
}

/**
 * @deprecated Use react/standalone_mount.tsx via main.tsx entry.
 */
export async function bootStandalone(_root: HTMLElement): Promise<void> {
  // Retained for type/export compatibility; main.tsx mounts React directly.
}

export type { ViewHandle };
