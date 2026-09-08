/** Shell nav toggle expanded state: desktop sidebar vs mobile sheet (class A dock). */
export function navigationExpanded(
  viewportIsMdUp: boolean,
  sidebarCollapsed: boolean,
  mobileNavOpen: boolean
): boolean {
  return viewportIsMdUp ? !sidebarCollapsed : mobileNavOpen;
}
