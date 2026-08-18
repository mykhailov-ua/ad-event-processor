import { useMemo } from 'react';
import { perfOverlayEnabled } from '../helpers/perf_display.js';
import { probeReport } from '../helpers/perf_probe.js';

export type PerfBlockProps = {
  id: string;
};

export function PerfBlock({ id }: PerfBlockProps) {
  const enabled = perfOverlayEnabled();
  const report = useMemo(() => (enabled ? JSON.stringify(probeReport(), null, 2) : ''), [enabled]);

  if (!enabled) return null;

  return (
    <pre id={id} aria-label="Critical path metrics">
      {report}
    </pre>
  );
}
