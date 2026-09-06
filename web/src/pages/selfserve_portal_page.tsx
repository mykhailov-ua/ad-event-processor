import { SelfServePortal } from '@/domains/portals/selfserve_portal';
import { useSelfServePortalPageWorkspace } from '@/domains/portals/use_selfserve_portal_page_workspace';

export function SelfServePortalPage() {
  return <SelfServePortal {...useSelfServePortalPageWorkspace()} />;
}
