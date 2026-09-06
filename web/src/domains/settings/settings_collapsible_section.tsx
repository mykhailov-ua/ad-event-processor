import { ChevronDown } from 'lucide-react';
import type { ReactNode } from 'react';

import {
  settingsCardClass,
  settingsCardTitleClass,
  settingsCollapsibleBodyClass,
  settingsCollapsibleSummaryClass,
} from '@/domains/settings/settings_classes';
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
    <details className={cn(settingsCardClass, 'group min-w-0')} open={defaultOpen || undefined}>
      <summary className={settingsCollapsibleSummaryClass}>
        <span className={settingsCardTitleClass}>{title}</span>
        <span className="flex items-center gap-2">
          {badge ? <Badge variant="outline">{badge}</Badge> : null}
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-open:rotate-180" />
        </span>
      </summary>
      <div className={settingsCollapsibleBodyClass}>{children}</div>
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
