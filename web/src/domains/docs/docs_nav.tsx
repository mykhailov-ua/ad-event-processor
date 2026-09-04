import { Link, useLocation } from 'react-router-dom';

import { DOCS_SECTIONS } from '@/lib/docs_sections';
import { cn } from '@/lib/utils';

export function DocsNav() {
  const location = useLocation();

  return (
    <nav aria-label="Documentation sections" className="flex min-w-0 flex-col gap-0.5">
      {DOCS_SECTIONS.map((section) => {
        const path = `/docs/${section.id}`;
        const active = location.pathname === path;
        return (
          <Link
            key={section.id}
            to={path}
            aria-current={active ? 'page' : undefined}
            className={cn(
              'block min-w-0 rounded-md px-3 py-2 text-sm leading-snug transition-colors',
              active
                ? 'bg-primary font-medium text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted/70 hover:text-foreground',
            )}
          >
            <span className="block break-words">{section.title}</span>
          </Link>
        );
      })}
    </nav>
  );
}
