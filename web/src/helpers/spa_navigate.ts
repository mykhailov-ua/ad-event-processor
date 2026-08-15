/** Registered by React shell (BrowserRouter navigate). */
let navigateHandler: ((path: string) => void) | null = null;

export function setSpaNavigate(handler: (path: string) => void): void {
  navigateHandler = handler;
}

/**
 * SPA navigation for imperative callers (command palette, legacy panels).
 */
export function navigate(path: string): void {
  if (navigateHandler) {
    navigateHandler(path);
    return;
  }
  window.location.assign(path);
}
