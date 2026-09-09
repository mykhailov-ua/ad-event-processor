import {
  TrackerShellHeaderActions,
  TrackerShellHeaderSearch,
  TrackerShellSidebarToggle,
} from '@/shell/tracker_shell_header';
import { adminSpacing } from '@/lib/admin_spacing';
import { PageBreadcrumbs } from '@/shell/page_breadcrumbs';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

export type TrackerHeaderBandProps = {
  navigationExpanded: boolean;
  onNavToggle: () => void;
  onOpenCommandPalette: () => void;
  variant?: 'd' | 'bare';
};

export function TrackerHeaderBand({
  navigationExpanded,
  onNavToggle,
  onOpenCommandPalette,
  variant = 'd',
}: TrackerHeaderBandProps) {
  if (variant === 'bare') {
    return (
      <div className={cn(shellChrome.trackerHeaderClass, 'justify-between')}>
        <TrackerShellSidebarToggle expanded={navigationExpanded} onToggle={onNavToggle} />
        <PageBreadcrumbs />
        <TrackerShellHeaderSearch onOpenCommandPalette={onOpenCommandPalette} />
        <TrackerShellHeaderActions />
      </div>
    );
  }

  return (
    <header className={cn(shellChrome.trackerHeaderClass, adminSpacing.inset.headerX)}>
      <div className={cn(adminSpacing.flex.headerStart, adminSpacing.gap.md)}>
        <TrackerShellSidebarToggle expanded={navigationExpanded} onToggle={onNavToggle} />
        <PageBreadcrumbs />
      </div>
      <div className={adminSpacing.flex.headerStart}>
        <TrackerShellHeaderSearch onOpenCommandPalette={onOpenCommandPalette} />
      </div>
      <div className={adminSpacing.flex.headerEnd}>
        <TrackerShellHeaderActions />
      </div>
    </header>
  );
}
