import { Link, useNavigate } from 'react-router-dom';

import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { Lander } from '@/api/types';
import {
  campaignListCellContentClass,
  campaignListCopyRowClass,
  campaignListCopyToolsSlotClass,
  campaignListEllipsisTextClass,
  campaignListNameRowCellClass,
  campaignListNameRowMenuSlotClass,
  campaignListNameRowTextClass,
  campaignListNameTextClass,
  campaignListTableFullWidthClass,
  campaignListTdClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  landerEditorPath,
  landerHostedCellText,
  landerHostedCellTitle,
  resolveLanderLiveUrl,
  resolveLanderPrimaryUrl,
} from '@/domains/creative/lander_list_helpers';
import { CopyButton } from '@/shell/copy_button';
import {
  DirectoryTable,
  DirectoryTableHead,
  directoryTableRevalidatingClass,
  TableBody,
  TableHeader,
} from '@/shell/directory_table';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { TableHost } from '@/shell/ui_bands';
import { displayRelativeTimestamp, displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';

const COLUMN_WIDTHS = {
  name: '20%',
  url: '50%',
  hosted: '18%',
  created: '12%',
} as const;

export type LandersListTableProps = {
  items: Lander[];
  fetching?: boolean;
  acting?: boolean;
  actingLanderId?: string | null;
  emptyMessage?: string;
  onOpenEditLander?: (lander: Lander) => void;
  onDeleteLander?: (lander: Lander) => void;
  onCopyUrl?: (url: string) => void;
};

export function LandersListTable({
  items,
  fetching = false,
  acting = false,
  actingLanderId = null,
  emptyMessage = 'No landers match the current filters.',
  onOpenEditLander,
  onDeleteLander,
  onCopyUrl,
}: LandersListTableProps) {
  const navigate = useNavigate();

  if (items.length === 0) {
    return (
      <TableHost className="w-full">
        <p className="p-4 text-[13px] text-muted-foreground">{emptyMessage}</p>
      </TableHost>
    );
  }

  return (
    <TableHost className="w-full">
      <DirectoryTable
        className={cn(
          'w-full rounded-none border-0 shadow-none',
          directoryTableRevalidatingClass(fetching)
        )}
        fixedLayout
        nested
        tableClassName={campaignListTableFullWidthClass}
        tableStyle={{
          width: '100%',
          tableLayout: 'fixed',
        }}
      >
        <colgroup>
          <col style={{ width: COLUMN_WIDTHS.name }} />
          <col style={{ width: COLUMN_WIDTHS.url }} />
          <col style={{ width: COLUMN_WIDTHS.hosted }} />
          <col style={{ width: COLUMN_WIDTHS.created }} />
        </colgroup>
        <TableHeader>
          <tr>
            <DirectoryTableHead>Name</DirectoryTableHead>
            <DirectoryTableHead>URL</DirectoryTableHead>
            <DirectoryTableHead>Hosted</DirectoryTableHead>
            <DirectoryTableHead>Created</DirectoryTableHead>
          </tr>
        </TableHeader>
        <TableBody>
          {items.map((row) => {
            const primaryUrl = resolveLanderPrimaryUrl(row);
            const liveUrl = resolveLanderLiveUrl(row);
            const rowActing = actingLanderId === row.id;

            return (
              <tr key={row.id}>
                <td className={campaignListTdClass}>
                  <div className={campaignListNameRowCellClass}>
                    <div className={campaignListNameRowTextClass}>
                      <Link
                        className={cn(
                          campaignListEllipsisTextClass,
                          campaignListNameTextClass,
                          'hover:underline'
                        )}
                        title={row.name}
                        to={landerEditorPath(row.id)}
                      >
                        {row.name}
                      </Link>
                    </div>
                    <div className={campaignListNameRowMenuSlotClass}>
                      <RowActionsMenu ariaLabel={`Actions for ${row.name}`} disabled={acting}>
                        <DropdownMenuItem onClick={() => navigate(landerEditorPath(row.id))}>
                          Open editor
                        </DropdownMenuItem>
                        {onOpenEditLander ? (
                          <DropdownMenuItem disabled={rowActing} onClick={() => onOpenEditLander(row)}>
                            Edit
                          </DropdownMenuItem>
                        ) : null}
                        {primaryUrl ? (
                          <DropdownMenuItem
                            onClick={() => {
                              window.open(primaryUrl, '_blank', 'noopener,noreferrer');
                            }}
                          >
                            Open URL
                          </DropdownMenuItem>
                        ) : null}
                        {liveUrl ? (
                          <DropdownMenuItem
                            onClick={() => {
                              window.open(liveUrl, '_blank', 'noopener,noreferrer');
                            }}
                          >
                            Open live
                          </DropdownMenuItem>
                        ) : null}
                        {primaryUrl && onCopyUrl ? (
                          <DropdownMenuItem disabled={rowActing} onClick={() => onCopyUrl(primaryUrl)}>
                            Copy URL
                          </DropdownMenuItem>
                        ) : null}
                        {onDeleteLander ? (
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            disabled={rowActing}
                            onClick={() => onDeleteLander(row)}
                          >
                            Delete
                          </DropdownMenuItem>
                        ) : null}
                      </RowActionsMenu>
                    </div>
                  </div>
                </td>
                <td className={campaignListTdClass}>
                  {primaryUrl ? (
                    <div className={campaignListCopyRowClass}>
                      <a
                        className={cn(
                          campaignListEllipsisTextClass,
                          'min-w-0 flex-1 text-[13px] text-foreground underline'
                        )}
                        href={primaryUrl}
                        rel="noreferrer"
                        target="_blank"
                        title={primaryUrl}
                      >
                        {primaryUrl}
                      </a>
                      <div className={campaignListCopyToolsSlotClass}>
                        <CopyButton label="URL" showToast={false} value={primaryUrl} />
                      </div>
                    </div>
                  ) : (
                    <span className={campaignListCellContentClass}>-</span>
                  )}
                </td>
                <td className={campaignListTdClass}>
                  <span className={campaignListCellContentClass} title={landerHostedCellTitle(row)}>
                    {landerHostedCellText(row)}
                  </span>
                </td>
                <td className={campaignListTdClass}>
                  <span
                    className={campaignListCellContentClass}
                    title={displayTimestamp(row.created_at)}
                  >
                    {displayRelativeTimestamp(row.created_at)}
                  </span>
                </td>
              </tr>
            );
          })}
        </TableBody>
      </DirectoryTable>
    </TableHost>
  );
}
