import { MoreHorizontal } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

import type { Campaign } from '@/api/types';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { directoryTableRowMenuButtonClass } from '@/shell/directory_table_row_actions';

export type CampaignListTableRowMenuProps = {
  campaign: Campaign;
  onOpenOverview?: (campaign: Campaign) => void;
};

export function CampaignListTableRowMenu({
  campaign,
  onOpenOverview,
}: CampaignListTableRowMenuProps) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          aria-label={`Actions for ${campaign.name}`}
          className={directoryTableRowMenuButtonClass}
          type="button"
          onClick={(event) => event.stopPropagation()}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem onSelect={() => onOpenOverview?.(campaign)}>Overview</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => navigate(`/campaigns/${campaign.id}/edit`)}>
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => navigate(`/dashboards/campaign/${campaign.id}`)}>
          View report
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
