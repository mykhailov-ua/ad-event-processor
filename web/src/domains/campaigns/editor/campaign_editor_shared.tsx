import type { ReactNode } from 'react';

import { ApiError } from '@/api/client';
import { Badge } from '@/components/ui/badge';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import type { CloneCampaignOptions } from '@/api/campaigns_api';
import { cloneMutationErrorMessage } from '@/domains/campaigns/editor/campaign_clone_request';
import { adminKit, adminSpacing, adminTypography } from '@/lib/admin_kit';
import { userErrorMessage } from '@/lib/admin_error';
import { cn } from '@/lib/utils';

/** Bordered editor/wizard section card. */
export const campaignEditorSectionClass = cn(
  `grid ${adminSpacing.gap.xl} border border-border bg-card ${adminSpacing.inset.panel}`,
  adminKit.panelRadius
);

/** Muted inset panel inside an editor section. */
export const campaignEditorInsetPanelClass = cn(
  `grid ${adminSpacing.gap.md} border border-border bg-muted ${adminSpacing.inset.panel}`,
  adminTypography.body,
  adminKit.panelRadius
);

/** Wizard/import panel stack inside sheet body. */
export const campaignEditorWizardRootClass = `grid ${adminSpacing.gap.xl}`;

/** Two-column form row with explicit horizontal gap between fields. */
export const campaignEditorFormColumnsClass = cn(
  'grid w-full min-w-0 grid-cols-1 sm:grid-cols-2',
  adminSpacing.gapX.formColumns,
  adminSpacing.gapY.formColumns
);

export const campaignEditorActionsRowClass = adminSpacing.flex.actionsRowEnd;

export function formatReadonly(value: string | undefined): string {
  if (value == null || value === '') {
    return '-';
  }
  return value;
}

function fieldErrorEntries(fieldErrors: Record<string, string> | undefined): [string, string][] {
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
    <div className={`grid ${adminSpacing.gap.md}`}>
      <p className={adminTypography.label}>{title}</p>
      <ul
        className={cn(
          'm-0 flex list-disc flex-col pl-5',
          adminSpacing.gap.xs,
          adminTypography.bodyMuted
        )}
      >
        {entries.map(([field, message]) => (
          <li key={field}>
            <span className={adminTypography.captionPlain}>{field}</span>: {message}
          </li>
        ))}
      </ul>
      <pre className={cn('overflow-x-auto bg-muted p-2', adminTypography.monoData)}>
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
    <Badge variant={valid ? 'secondary' : 'destructive'}>{valid ? validLabel : invalidLabel}</Badge>
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

export function diffSeverityVariant(severity: string): 'secondary' | 'destructive' | 'outline' {
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
    <div className={`grid ${adminSpacing.gap.xs}`}>
      <p className={adminTypography.label}>{title}</p>
      <ul
        className={cn(
          'm-0 flex list-disc flex-col pl-5',
          adminSpacing.gap.xs,
          adminTypography.bodyMuted
        )}
      >
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
        <StubBanner key="save" title="Save not available" message={userErrorMessage(saveError)} />
      ) : (
        <ErrorBlock key="save" title="Could not save campaign" error={saveError} />
      )
    );
  }
  if (publishCheckError) {
    blocks.push(
      <div key="publish-check">
        {editorApiErrorBlock(
          publishCheckError,
          'Publish check unavailable',
          'Could not check publish gate'
        )}
      </div>
    );
  }
  if (validateError) {
    blocks.push(
      <div key="validate">
        {editorApiErrorBlock(validateError, 'Validate unavailable', 'Could not validate changes')}
      </div>
    );
  }
  if (publishError) {
    blocks.push(
      <div key="publish">
        {editorApiErrorBlock(publishError, 'Publish unavailable', 'Could not publish campaign')}
      </div>
    );
  }
  if (blocks.length === 0) {
    return null;
  }
  return <div className={`grid ${adminSpacing.gap.lg}`}>{blocks}</div>;
}

export function editorApiErrorBlock(
  error: Error,
  stubTitle: string,
  errorTitle: string
): ReactNode {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={stubTitle} message={userErrorMessage(error)} />;
  }
  return <ErrorBlock title={errorTitle} error={error} />;
}

function clonePanelUserMessage(error: Error): string {
  const cloneMessage = cloneMutationErrorMessage(error);
  if (cloneMessage !== error.message) {
    return cloneMessage;
  }
  return userErrorMessage(error);
}

export function campaignPanelError(error: Error, title: string): ReactNode {
  if (/clone/i.test(title)) {
    if (error instanceof ApiError && error.status === 501) {
      return editorApiErrorBlock(error, `${title} unavailable`, title);
    }
    return <ErrorBlock title={title} error={error} message={clonePanelUserMessage(error)} />;
  }
  return editorApiErrorBlock(error, `${title} unavailable`, title);
}
