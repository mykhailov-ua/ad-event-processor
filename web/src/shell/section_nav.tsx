import { Link, useLocation } from 'react-router-dom';

import { isSectionNavActive, type SectionNavItem } from '@/lib/nav_config';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type SectionNavProps = {
  items: SectionNavItem[];
  label: string;
  variant?: 'pill' | 'admin';
};

export function SectionNav({ items, label, variant = 'pill' }: SectionNavProps) {
  const location = useLocation();

  return (
    <nav
      aria-label={label}
     
    >
      {items.map((item) => {
        const active = isSectionNavActive(location.pathname, item);
        if (variant === 'admin') {
          return (
            <Link
              key={item.path}
              to={item.path}
              aria-current={active ? 'page' : undefined}
             
            >
              {item.label}
            </Link>
          );
        }
        return (
          <Link
            key={item.path}
            to={item.path}
           
          >
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
