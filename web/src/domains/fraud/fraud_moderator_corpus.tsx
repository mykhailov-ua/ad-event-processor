import { Link } from 'react-router-dom';

import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { ModeratorCorpusTuple } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { FraudLimitsDocLink } from '@/domains/fraud/fraud_limits_doc_link';

export type FraudModeratorCorpusProps = {
  items?: ModeratorCorpusTuple[];
  total: number;
  limit: number;
  offset: number;
  lastRefresh?: string;
  fetching: boolean;
  revalidating?: boolean;
  error: Error | undefined;
  saving: boolean;
  saveError: Error | undefined;
  importing: boolean;
  importError: Error | undefined;
  previewCount?: number;
  previewError: Error | undefined;
  draftJa3: string;
  draftJa4: string;
  draftTcpSig: string;
  draftWebgl: string;
  draftDesync: string;
  draftNote: string;
  draftCsv: string;
  setDraftJa3: (value: string) => void;
  setDraftJa4: (value: string) => void;
  setDraftTcpSig: (value: string) => void;
  setDraftWebgl: (value: string) => void;
  setDraftDesync: (value: string) => void;
  setDraftNote: (value: string) => void;
  setDraftCsv: (value: string) => void;
  onSaveTuple: () => void;
  onImportCsv: () => void;
  onPreview: () => void;
  onLimitChange: (limit: number) => void;
  onOffsetChange: (offset: number) => void;
};

export function FraudModeratorCorpus(props: FraudModeratorCorpusProps) {
  if (props.fetching && !props.items) {
    return <PageSkeleton />;
  }

  const rows = props.items ?? [];
  const pageEnd = props.offset + props.limit;
  const hasPrev = props.offset > 0;
  const hasNext = pageEnd < props.total;

  return (
    <PageChrome
      description="Learned moderator JA3/JA4/TCP/WebGL tuples for review-traffic safe-page routing."
      title="Moderator corpus"
    >
      <div className="flex flex-col gap-6">
        <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
          <FraudLimitsDocLink />
          <Link className="underline-offset-4 hover:underline" to="/reports/layer-desync-summary">
            Layer desync reports
          </Link>
          {props.lastRefresh ? (
            <span>Feed refresh: {displayTimestamp(props.lastRefresh)}</span>
          ) : null}
        </div>

        {props.error ? (
          <ErrorBlock message={props.error.message} title="Failed to load corpus" />
        ) : null}

        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">Upsert tuple</h2>
          <div className="grid gap-3 md:grid-cols-2">
            <Input
              placeholder="JA3 (required)"
              value={props.draftJa3}
              onChange={(e) => props.setDraftJa3(e.target.value)}
            />
            <Input
              placeholder="JA4"
              value={props.draftJa4}
              onChange={(e) => props.setDraftJa4(e.target.value)}
            />
            <Input
              placeholder="TCP sig (hex)"
              value={props.draftTcpSig}
              onChange={(e) => props.setDraftTcpSig(e.target.value)}
            />
            <Input
              placeholder="WebGL renderer"
              value={props.draftWebgl}
              onChange={(e) => props.setDraftWebgl(e.target.value)}
            />
            <Input
              placeholder="Layer desync min count"
              value={props.draftDesync}
              onChange={(e) => props.setDraftDesync(e.target.value)}
            />
            <Input
              placeholder="Note"
              value={props.draftNote}
              onChange={(e) => props.setDraftNote(e.target.value)}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button disabled={props.saving} loading={props.saving} onClick={props.onSaveTuple}>
              Save tuple
            </Button>
            <Button type="button" variant="outline" onClick={props.onPreview}>
              Preview 7d matches
            </Button>
          </div>
          {props.saveError ? (
            <ErrorBlock message={props.saveError.message} title="Save failed" />
          ) : null}
          {props.previewError ? (
            <ErrorBlock message={props.previewError.message} title="Preview failed" />
          ) : null}
          {props.previewCount != null ? (
            <p className="text-sm text-muted-foreground">
              Review-routed clicks with matching JA3 in last 7 days: {props.previewCount}
            </p>
          ) : null}
        </section>

        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-medium">CSV import</h2>
          <Textarea
            className="min-h-28 text-xs"
            placeholder="ja3,ja4,tcp_sig,webgl_renderer,layer_desync_count,note"
            value={props.draftCsv}
            onChange={(e) => props.setDraftCsv(e.target.value)}
          />
          <Button disabled={props.importing} loading={props.importing} onClick={props.onImportCsv}>
            Import CSV
          </Button>
          {props.importError ? (
            <ErrorBlock message={props.importError.message} title="Import failed" />
          ) : null}
        </section>

        <section className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-sm font-medium">Corpus tuples</h2>
            <span className="text-sm text-muted-foreground">{props.total} total</span>
          </div>
          {rows.length === 0 ? (
            <EmptyState
              description="Import from layer-desync drilldown or add a tuple above."
              title="No corpus tuples"
            />
          ) : (
            <DirectoryTable>
              <DirectoryTableHead>
                <TableRow>
                  <TableHeader>JA3</TableHeader>
                  <TableHeader>JA4</TableHeader>
                  <TableHeader>TCP</TableHeader>
                  <TableHeader>WebGL</TableHeader>
                  <TableHeader>Desync</TableHeader>
                  <TableHeader>Source</TableHeader>
                  <TableHeader>Updated</TableHeader>
                </TableRow>
              </DirectoryTableHead>
              <TableBody>
                {rows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="text-xs">{row.ja3}</TableCell>
                    <TableCell className="text-xs">{row.ja4 || '-'}</TableCell>
                    <TableCell className="text-xs">{row.tcp_sig || '-'}</TableCell>
                    <TableCell>{row.webgl_renderer || '-'}</TableCell>
                    <TableCell>{row.layer_desync_count ?? 0}</TableCell>
                    <TableCell>{row.source || '-'}</TableCell>
                    <TableCell>
                      {displayTimestamp(row.updated_at_display ?? row.updated_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          )}
          <div className="flex gap-2">
            <Button
              disabled={!hasPrev}
              type="button"
              variant="outline"
              onClick={() => props.onOffsetChange(Math.max(0, props.offset - props.limit))}
            >
              Previous
            </Button>
            <Button
              disabled={!hasNext}
              type="button"
              variant="outline"
              onClick={() => props.onOffsetChange(props.offset + props.limit)}
            >
              Next
            </Button>
          </div>
        </section>
      </div>
    </PageChrome>
  );
}
