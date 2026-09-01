import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { createLander, listLanders } from '@/api/landers_api';
import { LandersDirectory } from '@/domains/creative/landers_directory';
import { useResource } from '@/hooks/use_resource';

export function LandersPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listLanders(signal),
    [reloadToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const items = useMemo(() => data ?? [], [data]);

  const onCreateLander = useCallback(async () => {
    const name = draftName.trim();
    if (!name) {
      setCreateError(new Error('Lander name is required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createLander({ name, url: draftUrl.trim() || undefined });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Lander created');
      setReloadToken((value) => value + 1);
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftName, draftUrl]);

  return (
    <LandersDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftName={draftName}
      draftUrl={draftUrl}
      creating={creating}
      createError={createError}
      createSuccess={createSuccess}
      onDraftNameChange={setDraftName}
      onDraftUrlChange={setDraftUrl}
      onCreateLander={() => {
        void onCreateLander();
      }}
    />
  );
}
