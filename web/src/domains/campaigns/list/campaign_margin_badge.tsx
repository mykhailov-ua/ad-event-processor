import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

export function CampaignMarginBreachBadge() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          aria-label="Margin guard breach"
          className="inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-destructive text-[10px] font-bold leading-none text-destructive-foreground"
          role="img"
        >
          !
        </span>
      </TooltipTrigger>
      <TooltipContent>
        Margin guard breach in the current reporting window
      </TooltipContent>
    </Tooltip>
  );
}
