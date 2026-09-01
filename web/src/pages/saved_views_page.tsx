import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createSavedView,
  deleteSavedView,
  listSavedViews,
  updateSavedView,
} from '@/api/saved_views_api';
import type { SavedView } from '@/api/types';
import { SavedViewsPanel } from '@/domains/portals/saved_views_panel';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

type EditRow = {
  name: string;
  report_key: string;
  spec_json: string;
};

function editRowFromView(row: SavedView): EditRow {
  return {
    name: row.name ?? '',
    report_key: row.report_key ?? '',
    spec_json: row.spec ? JSON.stringify(row.spec) : '',
  };
}

export function SavedViewsPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const [refreshToken, setRefreshToken] = useState(0);
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listSavedViews({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch],
  );

  const [draftName, setDraftName] = useState('');
  const [draftReportKey, setDraftReportKey] = useState('');
  const [draftSpecJson, setDraftSpecJson] = useState('{}');
  const [editRows, setEditRows] = useState<Record<string, EditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const views = useMemo(() => data ?? [], [data]);

  useEffect(() => {
    if (views.length === 0) {
      return;
    }
    setEditRows((current) => {
      const next = { ...current };
      for (const row of views) {
        const id = row.id ?? '';
        if (!id || next[id]) {
          continue;
        }
        next[id] = editRowFromView(row);
      }
      return next;
    });
  }, [views]);

  const parseSpec = useCallback((raw: string): Record<string, unknown> | undefined => {
    const trimmed = raw.trim();
    if (!trimmed) {
      return undefined;
    }
    const parsed: unknown = JSON.parse(trimmed);
    if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('Spec must be a JSON object');
    }
    return parsed as Record<string, unknown>;
  }, []);

  const onEditRowChange = useCallback(
    (id: string, field: 'name' | 'report_key' | 'spec_json', value: string) => {
      setEditRows((current) => ({
        ...current,
        [id]: {
          name: current[id]?.name ?? '',
          report_key: current[id]?.report_key ?? '',
          spec_json: current[id]?.spec_json ?? '',
          [field]: value,
        },
      }));
    },
    [],
  );

  const onCreateView = useCallback(async () => {
    const customerId = appliedCustomerId.trim();
    const name = draftName.trim();
    const reportKey = draftReportKey.trim();
    if (!customerId || !name || !reportKey) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setCreateSuccess(false);
    try {
      const spec = parseSpec(draftSpecJson);
      await createSavedView({
        customer_id: customerId,
        name,
        report_key: reportKey,
        spec,
      });
      setDraftName('');
      setDraftReportKey('');
      setDraftSpecJson('{}');
      setCreateSuccess(true);
      toast.success('Saved view created');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [appliedCustomerId, draftName, draftReportKey, draftSpecJson, parseSpec]);

  const onUpdateView = useCallback(
    async (id: string) => {
      const customerId = appliedCustomerId.trim();
      const edit = editRows[id];
      if (!customerId || !edit) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      try {
        const spec = parseSpec(edit.spec_json);
        await updateSavedView(id, {
          name: edit.name.trim(),
          report_key: edit.report_key.trim(),
          spec,
        });
        setRefreshToken((value) => value + 1);
      } catch (err) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setActing(false);
      }
    },
    [appliedCustomerId, editRows, parseSpec],
  );

  const onDeleteView = useCallback(async (id: string) => {
    setActing(true);
    setActionError(undefined);
    try {
      await deleteSavedView(id);
      setEditRows((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, []);

  return (
    <SavedViewsPanel
      views={views}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      draftName={draftName}
      draftReportKey={draftReportKey}
      draftSpecJson={draftSpecJson}
      editRows={editRows}
      acting={acting}
      actionError={actionError}
      createSuccess={createSuccess}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      onDraftNameChange={setDraftName}
      onDraftReportKeyChange={setDraftReportKey}
      onDraftSpecJsonChange={setDraftSpecJson}
      onEditRowChange={onEditRowChange}
      onCreateView={() => {
        void onCreateView();
      }}
      onUpdateView={(id) => {
        void onUpdateView(id);
      }}
      onDeleteView={(id) => {
        void onDeleteView(id);
      }}
    />
  );
}
