import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

export function CampaignMarginBreachBadge() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge className="shrink-0" variant="destructive">
          Margin
        </Badge>
      </TooltipTrigger>
      <TooltipContent>
        Margin guard breach in the current reporting window
      </TooltipContent>
    </Tooltip>
  );
}
