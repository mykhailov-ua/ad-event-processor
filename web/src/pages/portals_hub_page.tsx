import { useSession } from '@/hooks/use_session';
import { PortalsHub } from '@/domains/portals/portals_hub';

export function PortalsHubPage() {
  const { user } = useSession();
  return <PortalsHub permissions={user?.permissions} />;
}
