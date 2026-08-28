import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  buildDlqInboxUrl,
  buildDlqListUrl,
  retryDlq,
  retryDlqInbox,
  type DLQInbox,
  type DLQList,
} from '../helpers/ops_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsDlqPanel } from '../ui/ops/ops_dlq_panel.js';

const PAGE_LIMIT = 50;

export function OpsDlqPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const sourceFilter = searchParams.get('source') ?? '';
  const cursor = searchParams.get('cursor') ?? '';
  const [draftSource, setDraftSource] = useState(sourceFilter);
  const [retryBusyId, setRetryBusyId] = useState<string | null>(null);

  const dlqUrl = useMemo(
    () => buildDlqListUrl({ limit: PAGE_LIMIT, cursor: cursor || undefined }),
    [cursor]
  );
  const inboxUrl = useMemo(
    () =>
      buildDlqInboxUrl({
        limit: PAGE_LIMIT,
        cursor: cursor || undefined,
        source: sourceFilter || undefined,
      }),
    [cursor, sourceFilter]
  );

  const { data: dlq, loading: dlqLoading, error: dlqError, reload: reloadDlq } =
    useResource<DLQList>(dlqUrl);
  const {
    data: inbox,
    loading: inboxLoading,
    error: inboxError,
    reload: reloadInbox,
  } = useResource<DLQInbox>(inboxUrl);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') next.delete(key);
        else next.set(key, value);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onApplySource = useCallback(() => {
    patchParams({ source: draftSource.trim() || null, cursor: null });
  }, [draftSource, patchParams]);

  const onPrevCursor = useCallback(() => {
    patchParams({ cursor: null });
  }, [patchParams]);

  const onNextCursor = useCallback(() => {
    const next = inbox?.next_cursor ?? dlq?.next_cursor;
    if (!next) return;
    patchParams({ cursor: next });
  }, [dlq?.next_cursor, inbox?.next_cursor, patchParams]);

  const onRetryDlq = useCallback(
    (id: string) => {
      setRetryBusyId(id);
      void (async () => {
        try {
          await retryDlq(id);
          pushToastMessage({ title: 'DLQ retry enqueued', message: id });
          reloadDlq();
          reloadInbox();
        } catch (err) {
          if (err instanceof ConfirmCancelledError) return;
          pushToastMessage({
            title: 'DLQ retry failed',
            message: err instanceof Error ? err.message : 'Retry failed',
          });
        } finally {
          setRetryBusyId(null);
        }
      })();
    },
    [reloadDlq, reloadInbox]
  );

  const onRetryInbox = useCallback(
    (id: string, source: string) => {
      setRetryBusyId(id);
      void (async () => {
        try {
          await retryDlqInbox(id, source);
          pushToastMessage({ title: 'Inbox retry enqueued', message: id });
          reloadInbox();
        } catch (err) {
          if (err instanceof ConfirmCancelledError) return;
          pushToastMessage({
            title: 'Inbox retry failed',
            message: err instanceof Error ? err.message : 'Retry failed',
          });
        } finally {
          setRetryBusyId(null);
        }
      })();
    },
    [reloadInbox]
  );

  return (
    <OpsDlqPanel
      dlq={dlq}
      inbox={inbox}
      loading={dlqLoading || inboxLoading}
      error={dlqError ?? inboxError}
      sourceFilter={draftSource}
      onSourceFilterChange={setDraftSource}
      onApplySource={onApplySource}
      cursor={cursor}
      nextCursor={inbox?.next_cursor ?? dlq?.next_cursor ?? null}
      onPrevCursor={onPrevCursor}
      onNextCursor={onNextCursor}
      onRetryDlq={onRetryDlq}
      onRetryInbox={onRetryInbox}
      retryBusyId={retryBusyId}
    />
  );
}
