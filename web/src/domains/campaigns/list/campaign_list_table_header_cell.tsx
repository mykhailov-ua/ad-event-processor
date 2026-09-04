import { ArrowDown, ArrowUp, GripVertical } from 'lucide-react';
import { useState, type DragEvent, type PointerEvent, type ReactNode } from 'react';

import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  COLUMN_DRAG_MIME,
  isCampaignListNumericColumn,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListColDragGripClass,
  campaignListColGripClass,
  campaignListHeaderCellClass,
  campaignListHeaderLabelClass,
  campaignListHeaderLabelNumClass,
  campaignListHeaderToolsClass,
} from '@/domains/campaigns/list/campaign_list_classes';
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
  resizable?: boolean;
  resizeLabel?: string;
  onResizePointerDown?: (event: PointerEvent<HTMLDivElement>) => void;
  onDragStart: () => void;
  onDrop: (event: DragEvent<HTMLDivElement>) => void;
  onDragEnd: () => void;
};

export function CampaignListTableHeaderCell({
  columnId,
  appliedSort,
  appliedOrder,
  onColumnSort,
  disabled,
  draggable,
  resizable = false,
  resizeLabel,
  onResizePointerDown,
  onDragStart,
  onDrop,
  onDragEnd,
}: CampaignListTableHeaderCellProps) {
  const [dragOver, setDragOver] = useState(false);
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
        className={cn(
          'inline-flex max-w-full items-center gap-0.5 whitespace-nowrap',
          active && 'text-foreground',
        )}
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
      <span className="whitespace-nowrap" title={label}>
        {label}
      </span>
    );
  }

  const showTools = draggable || resizable;

  return (
    <div
      className={cn(campaignListHeaderCellClass, dragOver && 'bg-muted')}
      onDragEnd={() => {
        setDragOver(false);
        onDragEnd();
      }}
      onDragEnter={(event) => {
        if (!draggable) {
          return;
        }
        event.preventDefault();
        setDragOver(true);
      }}
      onDragLeave={(event) => {
        if (!draggable) {
          return;
        }
        if (event.currentTarget.contains(event.relatedTarget as Node)) {
          return;
        }
        setDragOver(false);
      }}
      onDragOver={(event) => {
        if (!draggable) {
          return;
        }
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        setDragOver(true);
      }}
      onDrop={(event) => {
        setDragOver(false);
        onDrop(event);
      }}
    >
      <div className={cn(isNum ? campaignListHeaderLabelNumClass : campaignListHeaderLabelClass)}>
        {labelNode}
      </div>
      {showTools ? (
        <div className={campaignListHeaderToolsClass}>
          {draggable ? (
            <span
              aria-label={`Reorder ${label} column`}
              className={campaignListColDragGripClass}
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
          className={campaignListColGripClass}
          data-col-resize=""
          role="separator"
          onPointerDown={onResizePointerDown}
        />
      ) : null}
    </div>
  );
}
