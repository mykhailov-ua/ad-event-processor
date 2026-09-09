import type { ReactNode } from 'react';

import type { ImportValidateJobRequest, MigratePullRequest } from '@/api/types';
import { cn } from '@/lib/utils';

export type SourceKind = ImportValidateJobRequest['source_kind'];
export type PullSourceKind = MigratePullRequest['source_kind'];

export const SOURCE_KINDS: SourceKind[] = [
  'keitaro_json',
  'keitaro_admin_api',
  'binom_json',
  'binom_report_api',
  'native_v1',
];

export const PULL_SOURCE_KINDS: PullSourceKind[] = ['keitaro_admin_api', 'binom_report_api'];

export function parsePayloadJson(raw: string): Record<string, unknown> | unknown[] {
  const parsed: unknown = JSON.parse(raw);
  if (parsed == null || (typeof parsed !== 'object' && !Array.isArray(parsed))) {
    throw new Error('Payload must be a JSON object or array.');
  }
  return parsed as Record<string, unknown> | unknown[];
}

export function ImportField({
  id,
  label,
  children,
}: {
  id: string;
  label: string;
  children: ReactNode;
}) {
  return (
    <label
     
      htmlFor={id}
    >
      {label}
      {children}
    </label>
  );
}
