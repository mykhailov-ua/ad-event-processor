import type { ReactNode } from 'react';

import { ApiError } from '@/api/client';
import { Badge } from '@/components/ui/badge';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import type { CloneCampaignOptions } from '@/api/campaigns_api';

export function formatReadonly(value: string | undefined): string {
  if (value == null || value === '') {
    return '-';
  }
  return value;
}

function fieldErrorEntries(
  fieldErrors: Record<string, string> | undefined,
): [string, string][] {
  if (!fieldErrors) {
    return [];
  }
  return Object.entries(fieldErrors);
}

export function FieldErrorsPanel({
  title,
  fieldErrors,
}: {
  title: string;
  fieldErrors: Record<string, string> | undefined;
}) {
  const entries = fieldErrorEntries(fieldErrors);
  if (entries.length === 0) {
    return null;
  }

  return (
    <div className="grid gap-2">
      <p className="text-sm font-medium">{title}</p>
      <ul className="list-inside list-disc text-sm text-muted-foreground">
        {entries.map(([field, message]) => (
          <li key={field}>
            <span className="font-mono text-xs">{field}</span>: {message}
          </li>
        ))}
      </ul>
      <pre className="overflow-x-auto rounded-md bg-muted p-2 text-xs">
        {JSON.stringify(fieldErrors, null, 2)}
      </pre>
    </div>
  );
}

export function ValidityBadge({
  valid,
  validLabel,
  invalidLabel,
}: {
  valid: boolean;
  validLabel: string;
  invalidLabel: string;
}) {
  return (
    <Badge variant={valid ? 'secondary' : 'destructive'}>
      {valid ? validLabel : invalidLabel}
    </Badge>
  );
}

export const CLONE_OPTION_FIELDS: {
  field: keyof CloneCampaignOptions;
  label: string;
  description: string;
}[] = [
  {
    field: 'include_flow',
    label: 'Include flow',
    description: 'Copy flow routing from the source campaign.',
  },
  {
    field: 'include_postbacks',
    label: 'Include postbacks',
    description: 'Copy postback and integration URLs.',
  },
  {
    field: 'include_fraud',
    label: 'Include fraud settings',
    description: 'Copy fraud presets and overrides.',
  },
  {
    field: 'include_placement_blocks',
    label: 'Include placement blocks',
    description: 'Copy blocked placement rules.',
  },
  {
    field: 'reset_spend',
    label: 'Reset spend',
    description: 'Start the clone with zero spend counters.',
  },
];

export function diffSeverityVariant(
  severity: string,
): 'secondary' | 'destructive' | 'outline' {
  if (severity === 'remove') {
    return 'destructive';
  }
  if (severity === 'add') {
    return 'secondary';
  }
  return 'outline';
}

export function StringList({ title, items }: { title: string; items: string[] | undefined }) {
  if (!items || items.length === 0) {
    return null;
  }

  return (
    <div className="grid gap-1">
      <p className="text-sm font-medium">{title}</p>
      <ul className="list-inside list-disc text-sm text-muted-foreground">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

export function EditorStatusBanners({
  saveError,
  publishCheckError,
  validateError,
  publishError,
}: {
  saveError: Error | undefined;
  publishCheckError: Error | undefined;
  validateError: Error | undefined;
  publishError: Error | undefined;
}): ReactNode {
  const blocks: ReactNode[] = [];
  if (saveError) {
    blocks.push(
      saveError instanceof ApiError && saveError.status === 501 ? (
        <StubBanner key="save" title="Save not available" message={saveError.message} />
      ) : (
        <ErrorBlock key="save" title="Could not save campaign" message={saveError.message} />
      ),
    );
  }
  if (publishCheckError) {
    blocks.push(
      <ErrorBlock
        key="publish-check"
        title="Could not check publish gate"
        message={publishCheckError.message}
      />,
    );
  }
  if (validateError) {
    blocks.push(
      <ErrorBlock key="validate" title="Could not validate changes" message={validateError.message} />,
    );
  }
  if (publishError) {
    blocks.push(
      <ErrorBlock key="publish" title="Could not publish campaign" message={publishError.message} />,
    );
  }
  if (blocks.length === 0) {
    return null;
  }
  return <div className="flex flex-col gap-3">{blocks}</div>;
}

export function editorApiErrorBlock(
  error: Error,
  stubTitle: string,
  errorTitle: string,
): ReactNode {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={stubTitle} message={error.message} />;
  }
  return <ErrorBlock title={errorTitle} message={error.message} />;
}
