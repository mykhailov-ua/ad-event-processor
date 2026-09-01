import { MoreHorizontal } from 'lucide-react';
import type { ReactNode } from 'react';

import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export type RowActionsMenuProps = {
  disabled?: boolean;
  ariaLabel?: string;
  children: ReactNode;
};

export function RowActionsMenu({
  disabled = false,
  ariaLabel = 'Row actions',
  children,
}: RowActionsMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          aria-label={ariaLabel}
          className="h-5 w-5 p-0"
          disabled={disabled}
          type="button"
          variant="ghost"
        >
          <MoreHorizontal aria-hidden="true" className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">{children}</DropdownMenuContent>
    </DropdownMenu>
  );
}
