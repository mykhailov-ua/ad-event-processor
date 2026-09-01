import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { cn } from '@/lib/utils';

type EmptyStateVariant = 'default' | 'no-results' | 'blank-slate';

type EmptyStateProps = {
  title?: string;
  description?: string;
  variant?: EmptyStateVariant;
  actionLabel?: string;
  actionHref?: string;
  onAction?: () => void;
};

const variantDefaults: Record<
  EmptyStateVariant,
  { title: string; description: string }
> = {
  default: {
    title: 'No records',
    description: 'No customers match the current query.',
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
}: EmptyStateProps) {
  const defaults = variantDefaults[variant];
  const resolvedTitle = title ?? defaults.title;
  const resolvedDescription = description ?? defaults.description;
  const showAction = Boolean(actionLabel && (actionHref || onAction));

  return (
    <div
      className={cn(
        'rounded-2xl border border-dashed border-border/40 bg-muted/20 p-8 text-center',
        variant === 'blank-slate' && 'bg-muted/30',
      )}
    >
      <p className="font-medium">{resolvedTitle}</p>
      <p className="mt-1 text-sm text-muted-foreground">{resolvedDescription}</p>
      {showAction ? (
        <div className="mt-4">
          {actionHref ? (
            <PrimaryActionButton asChild>
              <Link to={actionHref}>{actionLabel}</Link>
            </PrimaryActionButton>
          ) : (
            <PrimaryActionButton type="button" onClick={onAction}>
              {actionLabel}
            </PrimaryActionButton>
          )}
        </div>
      ) : null}
    </div>
  );
}
