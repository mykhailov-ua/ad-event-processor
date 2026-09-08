import { Link, useLocation } from 'react-router-dom';

import { isSectionNavActive, type SectionNavItem } from '@/lib/nav_config';
import { adminKit } from '@/lib/admin_kit';
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
      className={cn(
        variant === 'admin' ? 'flex flex-wrap gap-1' : 'flex flex-wrap gap-2',
        className
      )}
    >
      {items.map((item) => {
        const active = isSectionNavActive(location.pathname, item);
        if (variant === 'admin') {
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={active ? 'page' : undefined}
              className={cn(
                'inline-flex min-h-7 items-center border border-border bg-background py-1 text-[13px] leading-[18px] text-foreground',
                adminKit.controlPaddingX,
                adminKit.controlRadius,
                active && 'border-primary bg-primary text-primary-foreground'
              )}
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
                'inline-flex min-h-7 items-center border border-border bg-background py-1 text-[13px] leading-[18px] transition-colors',
                adminKit.controlPaddingX,
                adminKit.controlRadius,
                active
                  ? 'border-primary bg-primary font-medium text-primary-foreground'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'
              )}
          >
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
