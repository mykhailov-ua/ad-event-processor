import { TeamOverviewView } from '@/domains/team/team_overview';
import { useTeamPageWorkspace } from '@/domains/team/use_team_page_workspace';

export function TeamPage() {
  const workspace = useTeamPageWorkspace();

  return (
    <TeamOverviewView
      {...workspace}
      onSaveMember={(memberId) => void workspace.onSaveMember(memberId)}
    />
  );
}
