import type { ReactNode } from 'react';

import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { RowActionsMenu } from '@/shell/row_actions_menu';

export type DirectoryRowActionsMenuProps = {
  ariaLabel: string;
  disabled?: boolean;
  onOverview?: () => void;
  overviewLabel?: string;
  children?: ReactNode;
};

export function DirectoryRowActionsMenu({
  ariaLabel,
  disabled,
  onOverview,
  overviewLabel = 'Overview',
  children,
}: DirectoryRowActionsMenuProps) {
  return (
    <RowActionsMenu ariaLabel={ariaLabel} disabled={disabled}>
      {onOverview ? (
        <DropdownMenuItem disabled={disabled} onClick={onOverview}>
          {overviewLabel}
        </DropdownMenuItem>
      ) : null}
      {children}
    </RowActionsMenu>
  );
}
