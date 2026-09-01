import { ChevronDown } from 'lucide-react';
import type { ReactNode } from 'react';

import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

export function SettingsCollapsibleSection({
  badge,
  children,
  defaultOpen = false,
  title,
}: {
  badge?: ReactNode;
  children: ReactNode;
  defaultOpen?: boolean;
  title: string;
}) {
  return (
    <details className="ui-surface-raised group" open={defaultOpen || undefined}>
      <summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-5 py-4 marker:content-none [&::-webkit-details-marker]:hidden">
        <span className="text-base font-medium tracking-tight">{title}</span>
        <span className="flex items-center gap-2">
          {badge ? <Badge variant="outline">{badge}</Badge> : null}
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180" />
        </span>
      </summary>
      <div className={cn('border-t border-border/40 px-5 pb-5 pt-4')}>{children}</div>
    </details>
  );
}

export function formatJsonPayloadSize(payload: Record<string, unknown> | undefined): string {
  if (!payload) {
    return '0 B';
  }
  const bytes = new TextEncoder().encode(JSON.stringify(payload)).length;
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  return `${(bytes / 1024).toFixed(1)} KB`;
}
