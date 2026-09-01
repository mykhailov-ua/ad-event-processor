import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { createFlow, listFlows } from '@/api/flows_api';
import type { FlowPath } from '@/api/types';
import { FlowsDirectory } from '@/domains/creative/flows_directory';
import { useResource } from '@/hooks/use_resource';

export function FlowsPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listFlows(signal),
    [reloadToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftPathsJson, setDraftPathsJson] = useState(
    '[{"weight":100,"lander_id":"","offer_id":""}]',
  );
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const items = useMemo(() => data ?? [], [data]);

  const onCreateFlow = useCallback(async () => {
    const name = draftName.trim();
    if (!name) {
      setCreateError(new Error('Flow name is required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      const parsed: unknown = JSON.parse(draftPathsJson);
      if (!Array.isArray(parsed)) {
        throw new Error('Paths must be a JSON array.');
      }
      await createFlow({ name, paths: parsed as FlowPath[] });
      setCreateSuccess(true);
      setDraftName('');
      toast.success('Flow created');
      setReloadToken((value) => value + 1);
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftName, draftPathsJson]);

  return (
    <FlowsDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftName={draftName}
      draftPathsJson={draftPathsJson}
      creating={creating}
      createError={createError}
      createSuccess={createSuccess}
      onDraftNameChange={setDraftName}
      onDraftPathsJsonChange={setDraftPathsJson}
      onCreateFlow={() => {
        void onCreateFlow();
      }}
    />
  );
}
