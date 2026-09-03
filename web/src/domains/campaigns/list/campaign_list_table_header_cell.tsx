import { ArrowDown, ArrowUp } from 'lucide-react';
import type { DragEvent } from 'react';

import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  COLUMN_DRAG_MIME,
  isCampaignListNumericColumn,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import { sortFieldForCampaignColumn } from '@/domains/campaigns/list/campaign_list_sort';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import { cn } from '@/lib/utils';

export type CampaignListTableHeaderCellProps = {
  columnId: CampaignListColumnId;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  onColumnSort: (field: CampaignSortField) => void;
  disabled?: boolean;
  draggable: boolean;
  dragOver: boolean;
  onDragStart: () => void;
  onDragEnter: () => void;
  onDragOver: (event: DragEvent<HTMLTableCellElement>) => void;
  onDragLeave: () => void;
  onDrop: (event: DragEvent<HTMLTableCellElement>) => void;
  onDragEnd: () => void;
};

export function CampaignListTableHeaderCell({
  columnId,
  appliedSort,
  appliedOrder,
  onColumnSort,
  disabled,
  draggable,
  dragOver,
  onDragStart,
  onDragEnter,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
}: CampaignListTableHeaderCellProps) {
  const sortField = sortFieldForCampaignColumn(columnId);
  const active = sortField != null && appliedSort === sortField;
  const label = CAMPAIGN_LIST_COLUMN_LABELS[columnId];
  const isNum = isCampaignListNumericColumn(columnId);

  if (columnId === 'select') {
    return <span className="sr-only">Select</span>;
  }

  return (
    <>
      <div
        className={cn('flex w-full items-center gap-1', isNum && 'num', dragOver && 'bg-zinc-100 dark:bg-zinc-800')}
        onDragEnd={onDragEnd}
        onDragEnter={onDragEnter}
        onDragLeave={onDragLeave}
        onDragOver={onDragOver}
        onDrop={onDrop}
      >
        {sortField != null ? (
          <button
            className={cn('inline-flex items-center gap-0.5', isNum && 'text-right', active && 'text-blue-600 dark:text-blue-400')}
            disabled={disabled}
            title={label}
            type="button"
            onClick={() => onColumnSort(sortField)}
          >
            {label}
            {active ? (
              appliedOrder === 'asc' ? (
                <ArrowUp aria-hidden className="ml-0.5 h-3 w-3 shrink-0" />
              ) : (
                <ArrowDown aria-hidden className="ml-0.5 h-3 w-3 shrink-0" />
              )
            ) : null}
          </button>
        ) : (
          <span className={cn('truncate', isNum && 'text-right')} title={label}>
            {label}
          </span>
        )}
      </div>
      {draggable ? (
        <button
          aria-label={`Reorder ${label} column`}
          className="absolute right-0 top-0 z-10 h-full w-2 cursor-col-resize"
          data-col-resize=""
          draggable
          type="button"
          onDragStart={(event) => {
            event.dataTransfer.setData(COLUMN_DRAG_MIME, columnId);
            event.dataTransfer.effectAllowed = 'move';
            onDragStart();
          }}
        >
          ::
        </button>
      ) : null}
    </>
  );
}
