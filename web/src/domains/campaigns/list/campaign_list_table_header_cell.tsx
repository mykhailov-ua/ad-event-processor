import { ArrowDown, ArrowUp, GripVertical } from 'lucide-react';
import { useCallback, useRef, type DragEvent, type PointerEvent, type ReactNode } from 'react';

import { Button } from '@/components/ui/button';

import {
  CAMPAIGN_LIST_COLUMN_LABELS,
  COLUMN_DRAG_MIME,
  isCampaignListNumericColumn,
  type CampaignListColumnId,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  campaignListColDragGripClass,
  campaignListColResizeHandleClass,
  campaignListHeaderCellClass,
  campaignListHeaderCellNumClass,
  campaignListHeaderLabelClass,
  campaignListHeaderLabelNumClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  campaignListSortShowsAscIcon,
  sortFieldForCampaignColumn,
} from '@/domains/campaigns/list/campaign_list_sort';
import type { CampaignSortField, SortOrder } from '@/domains/campaigns/list/campaigns_list_types';
import { cn } from '@/lib/utils';

export type CampaignListTableHeaderCellProps = {
  columnId: CampaignListColumnId;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  onColumnSort: (field: CampaignSortField) => void;
  disabled?: boolean;
  draggable: boolean;
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
  onDragStart,
  onDrop,
  onDragEnd,
}: CampaignListTableHeaderCellProps) {
  const rootRef = useRef<HTMLDivElement>(null);

  const setDragOverHighlight = useCallback((active: boolean) => {
    const root = rootRef.current;
    if (!root) {
      return;
    }
    root.classList.toggle('bg-muted', active);
  }, []);

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
      <Button
        className={cn(
          'inline-flex h-auto max-w-full items-center gap-0.5 border-0 bg-transparent p-0 font-normal shadow-none hover:bg-transparent',
          isNum ? 'justify-end' : 'justify-start',
          active && 'text-foreground'
        )}
        disabled={disabled}
        title={label}
        type="button"
        variant="ghost"
        onClick={() => onColumnSort(sortField)}
      >
        {label}
        {active ? (
          campaignListSortShowsAscIcon(appliedOrder, isNum) ? (
            <ArrowUp aria-hidden className="h-3 w-3 shrink-0" />
          ) : (
            <ArrowDown aria-hidden className="h-3 w-3 shrink-0" />
          )
        ) : null}
      </Button>
    );
  } else {
    labelNode = (
      <span className="whitespace-nowrap" title={label}>
        {label}
      </span>
    );
  }

  const showTools = draggable;

  return (
    <div
      ref={rootRef}
      className={cn(
        isNum ? campaignListHeaderCellNumClass : campaignListHeaderCellClass,
        columnId === 'id' && 'px-2'
      )}
      onDragEnd={() => {
        setDragOverHighlight(false);
        onDragEnd();
      }}
      onDragEnter={(event) => {
        if (!draggable) {
          return;
        }
        event.preventDefault();
        setDragOverHighlight(true);
      }}
      onDragLeave={(event) => {
        if (!draggable) {
          return;
        }
        if (event.currentTarget.contains(event.relatedTarget as Node)) {
          return;
        }
        setDragOverHighlight(false);
      }}
      onDragOver={(event) => {
        if (!draggable) {
          return;
        }
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
        setDragOverHighlight(true);
      }}
      onDrop={(event) => {
        setDragOverHighlight(false);
        onDrop(event);
      }}
    >
      <div className={isNum ? campaignListHeaderLabelNumClass : campaignListHeaderLabelClass}>
        {labelNode}
      </div>
      {showTools && draggable ? (
        <span
          aria-label={`Reorder ${label} column`}
          className={campaignListColDragGripClass}
          data-col-grip=""
          draggable
          onDragStart={(event) => {
            event.dataTransfer.setData(COLUMN_DRAG_MIME, columnId);
            event.dataTransfer.effectAllowed = 'move';
            setDragOverHighlight(true);
            onDragStart();
          }}
        >
          <GripVertical aria-hidden className="h-3 w-3" />
        </span>
      ) : null}
    </div>
  );
}

export type CampaignListColumnResizeHandleProps = {
  label: string;
  onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
};

export function CampaignListColumnResizeHandle({
  label,
  onPointerDown,
}: CampaignListColumnResizeHandleProps) {
  return (
    <div
      aria-label={label}
      className={campaignListColResizeHandleClass}
      data-col-resize=""
      role="separator"
      onPointerDown={onPointerDown}
    />
  );
}
