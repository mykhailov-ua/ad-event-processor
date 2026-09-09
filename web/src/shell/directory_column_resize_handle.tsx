import type { PointerEvent } from 'react';

import { campaignListColResizeHandleClass } from '@/domains/campaigns/list/campaign_list_classes';

export type DirectoryColumnResizeHandleProps = {
  label: string;
  onPointerDown: (event: PointerEvent<HTMLDivElement>) => void;
};

export function DirectoryColumnResizeHandle({
  label,
  onPointerDown,
}: DirectoryColumnResizeHandleProps) {
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
