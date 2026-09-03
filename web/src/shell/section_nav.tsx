import { Link, useLocation } from 'react-router-dom';

import { isSectionNavActive, type SectionNavItem } from '@/lib/nav_config';
import { cn } from '@/lib/utils';

export type SectionNavProps = {
  items: SectionNavItem[];
  label: string;
  className?: string;
  variant?: 'pill' | 'admin';
};

export function SectionNav({ items, label, className, variant = 'pill' }: SectionNavProps) {
  const location = useLocation();

  return (
    <nav
      aria-label={label}
      className={cn(variant === 'admin' ? 'flex flex-wrap gap-1' : 'flex flex-wrap gap-2', className)}
    >
      {items.map((item) => {
        const active = isSectionNavActive(location.pathname, item);
        if (variant === 'admin') {
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={active ? 'page' : undefined}
              className={cn('inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-xs dark:border-zinc-700 dark:bg-zinc-900', active && 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900')}
            >
              {item.label}
            </Link>
          );
        }
        return (
          <Link
            key={item.path}
            to={item.path}
            className={cn(
              'rounded-sm px-3.5 py-1.5 text-sm transition-colors',
              active
                ? 'bg-secondary font-medium text-foreground'
                : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
            )}
          >
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
