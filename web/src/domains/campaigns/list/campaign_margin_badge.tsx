import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

export function CampaignMarginBreachBadge() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          aria-label="Margin guard breach"
         
          role="img"
        >
          !
        </span>
      </TooltipTrigger>
      <TooltipContent>Margin guard breach in the current reporting window</TooltipContent>
    </Tooltip>
  );
}
