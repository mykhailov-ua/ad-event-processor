import { useMemo } from 'react';
import { Menu } from 'lucide-react';
import { useLocation, useNavigate } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';
import type { NavGroup } from '@/lib/nav_config';
import { isSectionNavActive } from '@/lib/nav_config';
import { Badge } from '@/components/ui/badge';

export type HeaderNavMenuProps = {
  navGroups: NavGroup[];
  onOpenMobileNav: () => void;
};

function resolveCurrentLabel(pathname: string, navGroups: NavGroup[]): string {
  for (const group of navGroups) {
    for (const item of group.items) {
      if (isSectionNavActive(pathname, item)) {
        return item.label;
      }
    }
  }
  return 'Pages';
}

export function HeaderNavMenu({ navGroups, onOpenMobileNav }: HeaderNavMenuProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const currentLabel = useMemo(
    () => resolveCurrentLabel(location.pathname, navGroups),
    [location.pathname, navGroups]
  );

  return (
    <>
      <Button
        aria-haspopup="dialog"
        className="md:hidden"
        type="button"
        variant="outline"
        onClick={onOpenMobileNav}
      >
        <Menu aria-hidden className="h-4 w-4" />
        <span>{currentLabel}</span>
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-haspopup="menu"
            className="hidden md:inline-flex"
            type="button"
            variant="outline"
          >
            <Menu aria-hidden className="h-4 w-4" />
            <span>{currentLabel}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          className={cn(
            adminChrome.floating,
            'z-[10001] min-w-56 max-h-[min(28rem,calc(100vh-4rem))] overflow-auto'
          )}
        >
          {navGroups.map((group, groupIndex) => (
            <div key={group.id}>
              {groupIndex > 0 ? <DropdownMenuSeparator /> : null}
              <DropdownMenuLabel>{group.label}</DropdownMenuLabel>
              {group.items.map((item) => {
                const active = isSectionNavActive(location.pathname, item);
                return (
                  <DropdownMenuItem
                    key={item.path}
                    aria-current={active ? 'page' : undefined}
                    className={active ? 'bg-accent text-accent-foreground' : undefined}
                    onSelect={() => navigate(item.path)}
                  >
                    <span>{item.label}</span>
                    {item.badgeCount != null && item.badgeCount > 0 ? (
                      <Badge className="ml-auto" variant="secondary">
                        {item.badgeCount}
                      </Badge>
                    ) : null}
                  </DropdownMenuItem>
                );
              })}
            </div>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
