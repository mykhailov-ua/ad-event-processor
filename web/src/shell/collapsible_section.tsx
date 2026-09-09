import { ChevronDown } from 'lucide-react';
import { useState, type ReactNode } from 'react';

import { Button } from '@/components/ui/button';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type CollapsibleSectionProps = {
  title: ReactNode;
  children: ReactNode;
  defaultOpen?: boolean;
  className?: string;
};

export function CollapsibleSection({
  title,
  children,
  defaultOpen = false,
  className,
}: CollapsibleSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <section className={cn('grid', adminSpacing.gap.md, className)}>
      <Button
        aria-expanded={open}
        className={cn(
          'h-auto min-h-7 w-full justify-start gap-2 px-0 py-0 hover:bg-transparent',
          adminTypography.sectionTitle
        )}
        type="button"
        variant="ghost"
        onClick={() => setOpen((current) => !current)}
      >
        <ChevronDown
          aria-hidden
          className={cn('size-4 shrink-0 transition-transform', open && 'rotate-180')}
        />
        <span className="min-w-0 flex-1 text-left">{title}</span>
      </Button>
      {open ? <div className={cn('grid', adminSpacing.gap.md)}>{children}</div> : null}
    </section>
  );
}
