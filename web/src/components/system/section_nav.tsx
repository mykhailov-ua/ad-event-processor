import { Link, useLocation } from 'react-router-dom';

import { isSectionNavActive, type SectionNavItem } from '@/lib/nav_config';
import { cn } from '@/lib/utils';

export type SectionNavProps = {
  items: SectionNavItem[];
  label: string;
  className?: string;
};

export function SectionNav({ items, label, className }: SectionNavProps) {
  const location = useLocation();

  return (
    <nav aria-label={label} className={cn('flex flex-wrap gap-2', className)}>
      {items.map((item) => {
        const active = isSectionNavActive(location.pathname, item);
        return (
          <Link
            key={item.path}
            to={item.path}
            className={cn(
              'rounded-full px-3.5 py-1.5 text-sm transition-colors',
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
