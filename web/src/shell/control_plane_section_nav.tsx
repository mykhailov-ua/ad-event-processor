import { Link, useLocation } from 'react-router-dom';

import { isSectionNavActive, type SectionNavItem } from '@/lib/nav_config';

export type ControlPlaneSectionNavProps = {
  items: SectionNavItem[];
  label: string;
};

export function ControlPlaneSectionNav({ items, label }: ControlPlaneSectionNavProps) {
  const location = useLocation();

  return (
    <nav aria-label={label}>
      <ul>
        {items.map((item) => {
          const active = isSectionNavActive(location.pathname, item);
          return (
            <li key={item.path}>
              <Link aria-current={active ? 'page' : undefined} to={item.path}>
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
