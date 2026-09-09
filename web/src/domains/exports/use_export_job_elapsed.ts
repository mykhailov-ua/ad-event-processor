import { useEffect, useState } from 'react';

import { formatExportJobElapsed } from '@/domains/exports/export_hub_job_status';

export function useExportJobElapsed(active: boolean, startedAtMs: number | undefined): string {
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    if (!active || startedAtMs == null) {
      return;
    }
    setNowMs(Date.now());
    const timer = window.setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => {
      window.clearInterval(timer);
    };
  }, [active, startedAtMs]);

  if (!active || startedAtMs == null) {
    return '';
  }
  return formatExportJobElapsed(startedAtMs, nowMs);
}
