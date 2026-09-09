import { Pause, Play } from 'lucide-react';

import { directoryTableRowMenuButtonClass } from '@/shell/directory_table_row_actions';

export type CampaignStatusToggleProps = {
  disabled?: boolean;
  status: string;
  onPause: () => void;
  onResume: () => void;
};

function normalizeStatus(status: string): string {
  return status.trim().toUpperCase();
}

export function CampaignStatusToggle({
  disabled = false,
  status,
  onPause,
  onResume,
}: CampaignStatusToggleProps) {
  const normalized = normalizeStatus(status);
  if (normalized === 'ARCHIVED') {
    return null;
  }

  const paused = normalized === 'PAUSED';

  return (
    <button
     
      disabled={disabled}
      onClick={paused ? onResume : onPause}
      title={paused ? 'Resume campaign' : 'Pause campaign'}
      type="button"
    >
      {paused ? <Play  /> : <Pause  />}
      <span >{paused ? 'Resume campaign' : 'Pause campaign'}</span>
    </button>
  );
}
