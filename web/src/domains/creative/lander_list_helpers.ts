import type { Lander } from '@/api/types';

export type LanderHostingKind = 'external' | 'hosted' | 'unconfigured';

export function resolveLanderHostingKind(lander: Lander): LanderHostingKind {
  if (lander.hosted_asset_id) {
    return 'hosted';
  }
  if (lander.url?.trim()) {
    return 'external';
  }
  return 'unconfigured';
}

export function resolveLanderPrimaryUrl(lander: Lander): string {
  const hosted = lander.hosted_url?.trim();
  if (hosted) {
    return hosted;
  }
  return lander.url?.trim() ?? '';
}

export function resolveLanderLiveUrl(lander: Lander): string | undefined {
  if (!lander.hosted_asset_id) {
    return undefined;
  }
  const base = lander.hosted_url?.trim();
  if (!base) {
    return undefined;
  }
  return base.endsWith('/') ? `${base}index.html` : `${base}/index.html`;
}

export function landerEditorPath(landerId: string): string {
  return `/landers/${landerId}/editor`;
}

export function landerHostedCellText(lander: Lander): string {
  const kind = resolveLanderHostingKind(lander);
  if (kind === 'external') {
    return 'External';
  }
  if (kind === 'unconfigured') {
    return 'No URL';
  }
  const parts = ['Hosted'];
  if (lander.has_unpublished_draft) {
    parts.push('Draft');
  }
  if (lander.published_version && lander.published_version > 0) {
    parts.push(`v${lander.published_version}`);
  }
  return parts.join(' | ');
}

export function landerHostedCellTitle(lander: Lander): string | undefined {
  const label = landerHostedCellText(lander);
  const hostedUrl = lander.hosted_url?.trim();
  if (!hostedUrl) {
    return label;
  }
  return `${label} - ${hostedUrl}`;
}
