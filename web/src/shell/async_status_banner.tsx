import { useEffect } from 'react';

import { Loader2 } from 'lucide-react';

import { AdminError } from '@/shell/admin_error';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export type AsyncStatusPhase = 'idle' | 'pending' | 'ready' | 'error';

export type AsyncStatusBannerProps = {
  phase: AsyncStatusPhase;
  pendingLabel?: string;
  readyLabel?: string;
  error?: Error;
  readyDurationMs?: number;
  onReadyDismiss?: () => void;
};

export function AsyncStatusBanner({
  phase,
  pendingLabel = 'Preparing export...',
  readyLabel = 'Export ready',
  error,
  readyDurationMs = 3000,
  onReadyDismiss,
}: AsyncStatusBannerProps) {
  useEffect(() => {
    if (phase !== 'ready') {
      return;
    }
    const timer = window.setTimeout(() => {
      onReadyDismiss?.();
    }, readyDurationMs);
    return () => {
      window.clearTimeout(timer);
    };
  }, [onReadyDismiss, phase, readyDurationMs]);

  if (phase === 'idle') {
    return null;
  }

  if (phase === 'error' && error) {
    return (
      <div >
        <AdminError error={error} title="Export failed" variant="inline" />
      </div>
    );
  }

  if (phase === 'ready') {
    return (
      <div
        aria-live="polite"
       
        role="status"
      >
        <p >{readyLabel}</p>
      </div>
    );
  }

  return (
    <div
      aria-busy="true"
      aria-live="polite"
     
      role="status"
    >
      <Loader2  aria-hidden />
      <p >{pendingLabel}</p>
    </div>
  );
}
