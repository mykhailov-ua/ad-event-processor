import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { Flow, FlowPath } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';
import { displayTimestamp } from '@/lib/display';

export type FlowDetailProps = {
  flow: Flow | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName?: string;
  draftPathsJson?: string;
  saving?: boolean;
  saveError?: Error;
  onDraftNameChange?: (value: string) => void;
  onDraftPathsJsonChange?: (value: string) => void;
  onSaveFlow?: () => void;
};

function normalizePaths(paths: Flow['paths']): FlowPath[] {
  if (Array.isArray(paths)) {
    return paths;
  }
  return [];
}

export function FlowDetail({
  flow,
  fetching,
  error,
  hasSnapshot,
  draftName = '',
  draftPathsJson = '[{"weight":100,"landers":[],"offers":[]}]',
  saving = false,
  saveError,
  onDraftNameChange,
  onDraftPathsJsonChange,
  onSaveFlow,
}: FlowDetailProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Flow">
        <CreativeNav />
        {creativePanelError(error, 'Could not load flow')}
      </PageChrome>
    );
  }

  if (!flow) {
    return (
      <PageChrome title="Flow">
        <CreativeNav />
        {creativePanelError(new Error('Flow not found'), 'Could not load flow')}
      </PageChrome>
    );
  }

  const paths = normalizePaths(flow.paths);

  return (
    <PageChrome title={flow.name}>
      <CreativeNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/flows">
        Back to flows
      </Link>

      {onSaveFlow ? (
        <section className="grid gap-4">
          <h2 className="text-base font-semibold">Edit flow</h2>
          <div className="grid gap-2">
            <Label htmlFor="flow-edit-name">Name</Label>
            <Input
              id="flow-edit-name"
              placeholder="Flow name…"
              value={draftName}
              onChange={(event) => onDraftNameChange?.(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="flow-edit-paths">Paths JSON</Label>
            <Textarea
              id="flow-edit-paths"
              className="min-h-32 font-mono text-sm"
              placeholder='[{"weight":100,"landers":[],"offers":[]}]'
              value={draftPathsJson}
              onChange={(event) => onDraftPathsJsonChange?.(event.target.value)}
            />
          </div>
          {saveError ? creativePanelError(saveError, 'Could not save flow') : null}
          <div>
            <PrimaryActionButton loading={saving} onClick={onSaveFlow} type="button">
              Save flow
            </PrimaryActionButton>
          </div>
        </section>
      ) : null}

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Metadata</h2>
        <dl className="grid gap-1 text-sm">
          <div>
            <dt className="text-muted-foreground">Flow ID</dt>
            <dd className="font-mono text-xs">{flow.id}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Created</dt>
            <dd>{displayTimestamp(flow.created_at)}</dd>
          </div>
        </dl>
      </section>

      {paths.length > 0 ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Paths</h2>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Weight</TableHead>
                  <TableHead>Landers</TableHead>
                  <TableHead>Offers</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paths.map((path, index) => (
                  <TableRow key={`${path.weight}-${index}`}>
                    <TableCell>{path.weight}</TableCell>
                    <TableCell>{path.landers?.length ?? 0}</TableCell>
                    <TableCell>{path.offers?.length ?? 0}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </section>
      ) : null}

      <details className="grid gap-2">
        <summary className="cursor-pointer text-base font-semibold">Raw</summary>
        <pre className="overflow-x-auto rounded-xl border border-border/50 bg-muted/40 p-4 font-mono text-xs">
          {JSON.stringify(flow, null, 2)}
        </pre>
      </details>

      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
