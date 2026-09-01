import { Link, useLocation } from 'react-router-dom';

import { DOCS_SECTIONS } from '@/lib/docs_sections';
import { cn } from '@/lib/utils';

export function DocsNav() {
  const location = useLocation();

  return (
    <nav aria-label="Documentation sections" className="flex flex-col gap-1 text-sm">
      {DOCS_SECTIONS.map((section) => {
        const path = `/docs/${section.id}`;
        const active = location.pathname === path;
        return (
          <Link
            key={section.id}
            to={path}
            className={cn(
              'rounded-full px-3 py-2 transition-colors',
              active
                ? 'bg-secondary font-medium text-foreground'
                : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
            )}
          >
            {section.title}
          </Link>
        );
      })}
    </nav>
  );
}
