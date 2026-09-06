// L3 DLQ inbox: cursor_stack in URL encodes forward/back pagination; retry bumps refreshToken.
import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listDlqInbox, retryDlqInboxEntry } from '@/api/ops_api';
import type { DLQInboxEntry } from '@/api/types';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { parseListLimit } from '@/lib/list_query';

const CURSOR_STACK_KEY = 'cursor_stack';

function parseCursorStack(raw: string | null): string[] {
  if (!raw) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((value): value is string => typeof value === 'string');
  } catch {
    return [];
  }
}

export function useOpsDlqPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [retryingId, setRetryingId] = useState<string | undefined>();
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const limit = parseListLimit(searchParams.get('limit'), 200);
  const cursor = searchParams.get('cursor') ?? undefined;
  const cursorStack = useMemo(
    () => parseCursorStack(searchParams.get(CURSOR_STACK_KEY)),
    [searchParams]
  );

  const { data, error, fetching } = useResource(
    (signal) => listDlqInbox({ limit, cursor }, signal),
    [limit, cursor, refreshToken]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || retryingId != null);

  const updateCursors = useCallback(
    (nextCursor: string | undefined, nextStack: string[]) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      if (nextCursor) {
        next.set('cursor', nextCursor);
      } else {
        next.delete('cursor');
      }
      if (nextStack.length > 0) {
        next.set(CURSOR_STACK_KEY, JSON.stringify(nextStack));
      } else {
        next.delete(CURSOR_STACK_KEY);
      }
      setSearchParams(next, { replace: true });
    },
    [limit, searchParams, setSearchParams]
  );

  const onNext = useCallback(() => {
    if (!data?.next_cursor) {
      return;
    }
    const nextStack = cursor ? [...cursorStack, cursor] : cursorStack;
    updateCursors(data.next_cursor, nextStack);
  }, [cursor, cursorStack, data?.next_cursor, updateCursors]);

  const onPrev = useCallback(() => {
    if (cursorStack.length === 0) {
      updateCursors(undefined, []);
      return;
    }
    const nextStack = cursorStack.slice(0, -1);
    const previousCursor = cursorStack[cursorStack.length - 1];
    updateCursors(previousCursor, nextStack);
  }, [cursorStack, updateCursors]);

  const onRetry = useCallback(
    async (entry: DLQInboxEntry) => {
      if (!entry.id || !entry.source || retryingId != null) {
        return;
      }
      setRetryingId(entry.id);
      try {
        await retryDlqInboxEntry(entry.id, entry.source);
        bumpRefreshCoalesced();
      } finally {
        setRetryingId(undefined);
      }
    },
    [bumpRefreshCoalesced, retryingId]
  );

  return {
    items: data?.items,
    nextCursor: data?.next_cursor,
    partial: data?.partial,
    limit,
    fetching,
    error,
    hasSnapshot: data != null,
    retryingId,
    onPrev,
    onNext,
    canGoPrev: cursorStack.length > 0 || Boolean(cursor),
    onRetry,
  };
}
