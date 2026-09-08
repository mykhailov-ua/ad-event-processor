import { MoreHorizontal } from 'lucide-react';
import type { ReactNode } from 'react';

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { directoryTableRowMenuButtonClass } from '@/shell/directory_table_row_actions';

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
        <button
          aria-label={ariaLabel}
          className={directoryTableRowMenuButtonClass}
          disabled={disabled}
          type="button"
          onClick={(event) => event.stopPropagation()}
        >
          <MoreHorizontal aria-hidden="true" className="h-4 w-4" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">{children}</DropdownMenuContent>
    </DropdownMenu>
  );
}
