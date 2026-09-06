import { Inbox } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

type EmptyStateVariant = 'default' | 'no-results' | 'blank-slate';

type EmptyStateProps = {
  title?: string;
  description?: string;
  variant?: EmptyStateVariant;
  actionLabel?: string;
  actionHref?: string;
  onAction?: () => void;
  className?: string;
};

const variantDefaults: Record<EmptyStateVariant, { title: string; description: string }> = {
  default: {
    title: 'No data found',
    description: 'No records match the current view.',
  },
  'no-results': {
    title: 'No results',
    description: 'Nothing matches the current filters. Try adjusting or clearing them.',
  },
  'blank-slate': {
    title: 'Nothing here yet',
    description: 'Get started by creating your first record.',
  },
};

export function EmptyState({
  title,
  description,
  variant = 'default',
  actionLabel,
  actionHref,
  onAction,
  className,
}: EmptyStateProps) {
  const defaults = variantDefaults[variant];
  const resolvedTitle = title ?? defaults.title;
  const resolvedDescription = description ?? defaults.description;
  const showAction = Boolean(actionLabel && (actionHref || onAction));

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 border border-dashed border-border bg-muted/30 px-8 py-12 text-center',
        adminKit.panelRadius,
        className
      )}
    >
      <div
        className={cn(
          'flex h-12 w-12 items-center justify-center bg-background text-muted-foreground',
          adminKit.pillRadius
        )}
      >
        <Inbox aria-hidden className="h-6 w-6" />
      </div>
      <div className="flex max-w-md flex-col gap-1">
        <p className="m-0 text-base font-semibold text-foreground">{resolvedTitle}</p>
        <p className="m-0 text-[13px] leading-[18px] text-muted-foreground">
          {resolvedDescription}
        </p>
      </div>
      {showAction ? (
        actionHref ? (
          <Button asChild variant="default">
            <Link to={actionHref}>{actionLabel}</Link>
          </Button>
        ) : (
          <Button type="button" variant="default" onClick={onAction}>
            {actionLabel}
          </Button>
        )
      ) : null}
    </div>
  );
}
