import { useMemo } from 'react';
import { BUILD_LABEL } from '../lib/build_label.js';
import { Button } from './button.js';

const STORAGE_KEY = 'adminServerVersion';

export type VersionBannerProps = {
  serverVersion?: string | null;
};

/**
 * Reload prompt when the server version changes mid-session.
 */
export function VersionBanner({ serverVersion }: VersionBannerProps) {
  const content = useMemo(() => {
    const version = serverVersion?.trim() ?? '';
    if (!version) return null;

    let prev: string | null = null;
    try {
      prev = sessionStorage.getItem(STORAGE_KEY);
    } catch {
      prev = null;
    }

    try {
      sessionStorage.setItem(STORAGE_KEY, version);
    } catch {
      // ignore quota / private mode
    }

    if (!prev || prev === version) return null;

    const buildHint = BUILD_LABEL ? ` UI bundle ${BUILD_LABEL}.` : '';
    return {
      prev,
      version,
      message: `Server updated (${prev} → ${version}).${buildHint} Reload to pick up changes.`,
    };
  }, [serverVersion]);

  if (!content) return null;

  return (
    <div
      className="stub-banner mb-4 cluster cluster--sm items-center"
      style={{ borderColor: 'var(--warning)' }}
    >
      <span>{content.message}</span>
      <Button label="Reload" variant="secondary" size="sm" onClick={() => window.location.reload()} />
    </div>
  );
}
