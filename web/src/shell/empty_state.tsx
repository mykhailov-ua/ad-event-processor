import { Inbox } from 'lucide-react';
import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
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

const variantDefaults: Record<
  EmptyStateVariant,
  { title: string; description: string }
> = {
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
    <div className={cn('admin-empty-state', className)}>
      <div className="admin-empty-state__icon">
        <Inbox aria-hidden className="h-6 w-6" />
      </div>
      <p className="admin-empty-state__title">{resolvedTitle}</p>
      <p className="admin-empty-state__description">{resolvedDescription}</p>
      {showAction ? (
        <div className="mt-4">
          {actionHref ? (
            <Button asChild variant="default">
              <Link to={actionHref}>{actionLabel}</Link>
            </Button>
          ) : (
            <Button type="button" variant="default" onClick={onAction}>
              {actionLabel}
            </Button>
          )}
        </div>
      ) : null}
    </div>
  );
}
