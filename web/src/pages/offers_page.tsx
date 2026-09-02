import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { createOffer, listOffers } from '@/api/offers_api';
import { OffersDirectory } from '@/domains/creative/offers_directory';
import { useResource } from '@/api/use_resource';

export function OffersPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listOffers(signal),
    [reloadToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const items = useMemo(() => data ?? [], [data]);

  const onCreateOffer = useCallback(async () => {
    const name = draftName.trim();
    const url = draftUrl.trim();
    if (!name || !url) {
      setCreateError(new Error('Name and URL are required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createOffer({ name, url });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Offer created');
      setReloadToken((value) => value + 1);
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftName, draftUrl]);

  return (
    <OffersDirectory
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
      onCreateOffer={() => {
        void onCreateOffer();
      }}
    />
  );
}
