// L3 RTB deals directory: list + inline create; license gate via rtbLicenseGated on errors.
import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

import { createRtbDeal, listRtbDeals } from '@/api/rtb_api';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';

export function useRtbDealsPageWorkspace() {
  const navigate = useNavigate();
  const { session } = useSession();

  const [draftDealId, setDraftDealId] = useState('');
  const [draftCustomerId, setDraftCustomerId] = useState(session?.default_customer_id ?? '');
  const [draftFloorMicro, setDraftFloorMicro] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>(undefined);
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource((signal) => listRtbDeals(signal), [refreshToken]);

  const licenseGated = rtbLicenseGated(error);
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || creating);

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
      bumpRefreshCoalesced();
      toast.success('Deal created');
      if (created.id != null) {
        void navigate(`/rtb/deals/${created.id}`);
      }
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, draftCustomerId, draftDealId, draftFloorMicro, navigate]);

  return {
    items: data,
    fetching,
    error: licenseGated ? undefined : error,
    hasSnapshot: data != null || licenseGated,
    licenseGated,
    draftDealId,
    draftCustomerId,
    draftFloorMicro,
    creating,
    createError,
    onDraftDealIdChange: setDraftDealId,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftFloorMicroChange: setDraftFloorMicro,
    onCreateDeal: () => {
      void onCreateDeal();
    },
  };
}
