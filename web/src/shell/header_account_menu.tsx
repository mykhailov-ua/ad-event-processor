import { LogOut, Settings, User } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

const accountMenuItemClass = cn(adminChrome.menuItem, 'gap-2');

export type HeaderAccountMenuProps = {
  signingOut?: boolean;
  onSignOut: () => void;
};

export function HeaderAccountMenu({ signingOut = false, onSignOut }: HeaderAccountMenuProps) {
  const navigate = useNavigate();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button aria-haspopup="menu" type="button" variant="outline">
          <User aria-hidden className="h-4 w-4" />
          <span>Account</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className={cn(adminChrome.floating, 'z-[10001] min-w-44')}>
        <DropdownMenuItem className={accountMenuItemClass} onSelect={() => navigate('/settings')}>
          <Settings aria-hidden className="h-4 w-4 shrink-0 opacity-80" />
          Settings
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className={cn(accountMenuItemClass, 'text-destructive hover:text-destructive')}
          disabled={signingOut}
          onSelect={() => onSignOut()}
        >
          <LogOut aria-hidden className="h-4 w-4 shrink-0 opacity-80" />
          {signingOut ? 'Logging out...' : 'Logout'}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
