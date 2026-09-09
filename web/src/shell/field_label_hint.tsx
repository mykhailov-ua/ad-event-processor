import type { ReactNode } from 'react';
import { Info } from 'lucide-react';

import { Label } from '@/components/ui/label';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { DIRECTORY_FIELD_LABEL_CLASS } from '@/shell/filter_panel_classes';
import { cn } from '@/lib/utils';

export type FieldLabelWithHintProps = {
  htmlFor?: string;
  label: string;
  className?: string;
  hintAriaLabel?: string;
  children: ReactNode;
};

export function FieldLabelWithHint({
  htmlFor,
  label,
  className,
  hintAriaLabel,
  children,
}: FieldLabelWithHintProps) {
  return (
    <div className="flex min-h-[18px] items-center gap-1.5">
      <Label className={cn(DIRECTORY_FIELD_LABEL_CLASS, className)} htmlFor={htmlFor}>
        {label}
      </Label>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            aria-label={hintAriaLabel ?? `${label} help`}
            className="inline-flex size-4 shrink-0 items-center justify-center rounded-full text-muted-foreground hover:text-foreground"
            type="button"
          >
            <Info aria-hidden className="size-3.5" />
          </button>
        </TooltipTrigger>
        <TooltipContent
          align="start"
          className="pointer-events-none max-w-[22rem] whitespace-normal p-3 text-left text-[12px] leading-5"
          side="top"
        >
          {children}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
