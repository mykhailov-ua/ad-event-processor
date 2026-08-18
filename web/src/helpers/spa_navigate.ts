let navigateHandler: ((path: string) => void) | null = null;

export function setSpaNavigate(handler: (path: string) => void): void {
  navigateHandler = handler;
}

export function navigate(path: string): void {
  if (navigateHandler) {
    navigateHandler(path);
    return;
  }
  window.location.assign(path);
}
