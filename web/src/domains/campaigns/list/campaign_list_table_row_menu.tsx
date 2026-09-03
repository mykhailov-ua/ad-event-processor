import { MoreHorizontal } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

import type { Campaign } from '@/api/types';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export type CampaignListTableRowMenuProps = {
  campaign: Campaign;
  onOpenOverview?: (campaign: Campaign) => void;
};

export function CampaignListTableRowMenu({ campaign, onOpenOverview }: CampaignListTableRowMenuProps) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-label={`Actions for ${campaign.name}`}
          className="h-7 w-7 shrink-0"
          size="icon"
          type="button"
          variant="ghost"
          onClick={(event) => event.stopPropagation()}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuItem onSelect={() => onOpenOverview?.(campaign)}>Overview</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => navigate(`/campaigns/${campaign.id}/edit`)}>
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => navigate(`/dashboards/campaign/${campaign.id}`)}>
          Report
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
