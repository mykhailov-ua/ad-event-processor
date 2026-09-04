import { ArrowDown, ArrowUp, GripVertical } from 'lucide-react';
import type { DragEvent, PointerEvent, ReactNode } from 'react';

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
  resizable?: boolean;
  resizeLabel?: string;
  onResizePointerDown?: (event: PointerEvent<HTMLDivElement>) => void;
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
  resizable = false,
  resizeLabel,
  onResizePointerDown,
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

  let labelNode: ReactNode;
  if (sortField != null) {
    labelNode = (
      <button
        className={cn('inline-flex max-w-full items-center gap-0.5 truncate', active && 'text-foreground')}
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
    );
  } else {
    labelNode = (
      <span className="truncate" title={label}>
        {label}
      </span>
    );
  }

  const showTools = draggable || resizable;

  return (
    <div
      className={cn('campaign-table-header-cell', dragOver && 'bg-muted')}
      onDragEnd={onDragEnd}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      <div className={cn('campaign-table-header-cell__label', isNum && 'num')}>{labelNode}</div>
      {showTools ? (
        <div className="campaign-table-header-tools">
          {draggable ? (
            <span
              aria-label={`Reorder ${label} column`}
              className="campaign-table-col-drag-grip"
              data-col-grip=""
              draggable
              onDragStart={(event) => {
                event.dataTransfer.setData(COLUMN_DRAG_MIME, columnId);
                event.dataTransfer.effectAllowed = 'move';
                onDragStart();
              }}
            >
              <GripVertical aria-hidden className="h-3 w-3" />
            </span>
          ) : null}
        </div>
      ) : null}
      {resizable ? (
        <div
          aria-label={resizeLabel ?? `Resize ${label} column`}
          className="campaign-table-col-grip"
          data-col-resize=""
          role="separator"
          onPointerDown={onResizePointerDown}
        />
      ) : null}
    </div>
  );
}
