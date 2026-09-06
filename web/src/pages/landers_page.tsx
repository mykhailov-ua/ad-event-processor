import { LandersDirectory } from '@/domains/creative/landers_directory';
import { useLandersPageWorkspace } from '@/domains/creative/use_landers_page_workspace';

export function LandersPage() {
  return <LandersDirectory {...useLandersPageWorkspace()} />;
}
