import { Pause, Play } from 'lucide-react';

import { Button } from '@/components/ui/button';

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
    <Button
      className="size-7 shrink-0 p-0"
      disabled={disabled}
      onClick={paused ? onResume : onPause}
      title={paused ? 'Resume campaign' : 'Pause campaign'}
      type="button"
      variant="ghost"
    >
      {paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
      <span className="sr-only">{paused ? 'Resume campaign' : 'Pause campaign'}</span>
    </Button>
  );
}
