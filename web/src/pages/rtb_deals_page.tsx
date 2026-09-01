import { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

import { createRtbDeal, listRtbDeals } from '@/api/rtb_api';
import { RtbDealsDirectory } from '@/domains/rtb/rtb_deals_directory';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useResource } from '@/hooks/use_resource';
import { useSession } from '@/hooks/use_session';

export function RtbDealsPage() {
  const navigate = useNavigate();
  const { session } = useSession();

  const [draftDealId, setDraftDealId] = useState('');
  const [draftCustomerId, setDraftCustomerId] = useState(session?.default_customer_id ?? '');
  const [draftFloorMicro, setDraftFloorMicro] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>(undefined);
  const [reloadKey, setReloadKey] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => listRtbDeals(signal),
    [reloadKey],
  );

  const items = useMemo(() => data ?? [], [data]);
  const licenseGated = rtbLicenseGated(error);

  const onCreateDeal = useCallback(async () => {
    const dealId = draftDealId.trim();
    const customerId = draftCustomerId.trim();
    if (!dealId || !customerId) {
      setCreateError(new Error('deal_id and customer_id are required'));
      return;
    }

    setCreating(true);
    setCreateError(undefined);
    try {
      const floorMicro = draftFloorMicro.trim() ? Number(draftFloorMicro.trim()) : undefined;
      const created = await createRtbDeal({
        deal_id: dealId,
        customer_id: customerId,
        floor_micro: floorMicro,
      });
      setReloadKey((value) => value + 1);
      toast.success('Deal created');
      if (created.id != null) {
        void navigate(`/rtb/deals/${created.id}`);
      }
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftCustomerId, draftDealId, draftFloorMicro, navigate]);

  return (
    <RtbDealsDirectory
      items={items}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={data != null || licenseGated}
      licenseGated={licenseGated}
      draftDealId={draftDealId}
      draftCustomerId={draftCustomerId}
      draftFloorMicro={draftFloorMicro}
      creating={creating}
      createError={createError}
      onDraftDealIdChange={setDraftDealId}
      onDraftCustomerIdChange={setDraftCustomerId}
      onDraftFloorMicroChange={setDraftFloorMicro}
      onCreateDeal={() => {
        void onCreateDeal();
      }}
    />
  );
}
