import { Link } from 'react-router-dom';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { FilterField } from '@/shell/filter_panel';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import type { Flow, Lander, Offer } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { FlowEditorVisual } from '@/domains/creative/flow_editor_visual';
import {
  buildFlowClickPreviewUrl,
  validateVisualPathWeights,
  visualRowsToFlowPaths,
  type FlowPathVisualRow,
} from '@/domains/creative/flow_path_model';
import { flowPathsToJson } from '@/domains/creative/flow_editor_form';
import { displayTimestamp } from '@/lib/display';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { ActionLinksBand, MetaLinksBand } from '@/shell/ui_bands';
import { ErrorBlock } from '@/shell/error_block';

export type FlowDetailProps = {
  flow: Flow | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  landers: Lander[];
  offers: Offer[];
  draftName?: string;
  draftRows?: FlowPathVisualRow[];
  showAdvancedJson?: boolean;
  saving?: boolean;
  saveError?: Error;
  deleting?: boolean;
  deleteError?: Error;
  cloning?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftRowsChange?: (rows: FlowPathVisualRow[]) => void;
  onShowAdvancedJsonChange?: (value: boolean) => void;
  onSaveFlow?: () => void;
  onCloneFlow?: () => void;
  onDeleteFlow?: () => void;
};

export function FlowDetail({
  flow,
  fetching,
  error,
  hasSnapshot,
  landers,
  offers,
  draftName = '',
  draftRows = [],
  showAdvancedJson = false,
  saving = false,
  saveError,
  deleting = false,
  deleteError,
  cloning = false,
  onDraftNameChange,
  onDraftRowsChange,
  onShowAdvancedJsonChange,
  onSaveFlow,
  onCloneFlow,
  onDeleteFlow,
}: FlowDetailProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Flow">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load flow')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  if (!flow) {
    return (
      <PageChrome title="Flow">
        <CreativeDirectoryStack>
          {creativePanelError(new Error('Flow not found'), 'Could not load flow')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  const validationError = validateVisualPathWeights(draftRows);
  const previewUrl = buildFlowClickPreviewUrl(undefined, flow.id);

  return (
    <PageChrome title={flow.name}>
      <CreativeDirectoryStack>
        <MetaLinksBand>
          <Link to="/flows">Back to flows</Link>
        </MetaLinksBand>

        {onSaveFlow ? (
          <section className="grid gap-4">
            <h2 className="text-base font-semibold">Stream editor</h2>
            <FilterField htmlFor="flow-edit-name" label="Name">
              <Input
                id="flow-edit-name"
                placeholder="Flow name..."
                value={draftName}
                onChange={(event) => onDraftNameChange?.(event.target.value)}
              />
            </FilterField>

            <FlowEditorVisual
              disabled={saving}
              landers={landers}
              offers={offers}
              rows={draftRows}
              validationError={validationError ?? undefined}
              onRowsChange={(rows) => onDraftRowsChange?.(rows)}
            />

            <div className="grid gap-2 rounded-md border border-border p-3">
              <p className="text-sm font-medium">Click URL preview</p>
              <p className="text-xs break-all text-muted-foreground">{previewUrl}</p>
              <p className="text-xs text-muted-foreground">
                Attach this flow to a campaign (flow_id) before sending live traffic. The tracker
                resolves landers from the campaign flow snapshot on /click.
              </p>
            </div>

            <div className="flex items-center gap-2">
              <Checkbox
                checked={showAdvancedJson}
                id="flow-advanced-json"
                onCheckedChange={(checked) => onShowAdvancedJsonChange?.(checked === true)}
              />
              <Label className="font-normal" htmlFor="flow-advanced-json">
                Show advanced JSON
              </Label>
            </div>

            {showAdvancedJson ? (
              <Textarea
                readOnly
                className="min-h-32 text-sm"
                value={flowPathsToJson(visualRowsToFlowPaths(draftRows))}
              />
            ) : null}

            {saveError ? (
              <ErrorBlock message={saveError.message} title="Could not save flow" />
            ) : null}
            <ActionLinksBand>
              <PrimaryActionButton loading={saving} onClick={onSaveFlow} type="button">
                Save flow
              </PrimaryActionButton>
              {onCloneFlow ? (
                <SecondaryActionButton loading={cloning} onClick={onCloneFlow} type="button">
                  Clone flow
                </SecondaryActionButton>
              ) : null}
              {onDeleteFlow ? (
                <SecondaryActionButton loading={deleting} onClick={onDeleteFlow} type="button">
                  Delete flow
                </SecondaryActionButton>
              ) : null}
            </ActionLinksBand>
            {deleteError ? creativePanelError(deleteError, 'Could not delete flow') : null}
          </section>
        ) : null}

        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Metadata</h2>
          <dl className="grid gap-1 text-sm">
            <div>
              <dt className="text-muted-foreground">Flow ID</dt>
              <dd className="text-xs">{flow.id}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Created</dt>
              <dd>{displayTimestamp(flow.created_at)}</dd>
            </div>
          </dl>
        </section>

        <details className="grid gap-2">
          <summary className="cursor-pointer text-base font-semibold">Raw</summary>
          <pre className={cn('ui-code-block overflow-x-auto', adminKit.panelRadius)}>
            {JSON.stringify(flow, null, 2)}
          </pre>
        </details>

        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
