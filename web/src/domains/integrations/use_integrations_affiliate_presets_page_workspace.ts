// affiliate status presets: catalog GET on mount; apply imports preset then POST schema apply.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { applyAffiliateStatusPreset, listAffiliateStatusPresets } from '@/api/integrations_api';
import type { ApplyIntegrationSchemaResponse } from '@/api/types';
import { mutationError } from '@/lib/mutation_audit';
import {
  type AdminValidationError,
  requireNonEmpty,
  toastValidationError,
} from '@/lib/admin_validation_error';
import { useResource } from '@/api/use_resource';

export function useIntegrationsAffiliatePresetsPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => listAffiliateStatusPresets(signal), []);
  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [applyingPreset, setApplyingPreset] = useState<string | undefined>();
  const [applyError, setApplyError] = useState<Error | undefined>();
  const [applyResult, setApplyResult] = useState<ApplyIntegrationSchemaResponse | undefined>();
  const [formValidationError, setFormValidationError] = useState<
    AdminValidationError | undefined
  >();

  const onDraftCampaignIdChange = useCallback((value: string) => {
    setFormValidationError(undefined);
    setDraftCampaignId(value);
  }, []);

  const onApplyPreset = useCallback(
    async (presetName: string) => {
      const presetCheck = requireNonEmpty(presetName, 'Preset name', 'preset_name');
      if (!presetCheck.ok) {
        setFormValidationError(presetCheck.error);
        toastValidationError(presetCheck.error);
        return;
      }
      const campaignIdCheck = requireNonEmpty(draftCampaignId, 'Campaign ID', 'campaign_id');
      if (!campaignIdCheck.ok) {
        setFormValidationError(campaignIdCheck.error);
        toastValidationError(campaignIdCheck.error);
        return;
      }
      const campaignId = campaignIdCheck.value;
      setApplyingPreset(presetName);
      setApplyError(undefined);
      setApplyResult(undefined);
      try {
        const result = await applyAffiliateStatusPreset(presetCheck.value, campaignId);
        setApplyResult(result);
        const count = result.mappings_applied_count;
        toast.success(
          count != null
            ? `Applied ${presetName}: ${count} conversion mapping(s)`
            : `Applied ${presetName} to campaign`
        );
      } catch (err: unknown) {
        setApplyError(mutationError(err));
      } finally {
        setApplyingPreset(undefined);
      }
    },
    [draftCampaignId]
  );

  return {
    presets: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftCampaignId,
    onDraftCampaignIdChange,
    applyingPreset,
    applyError,
    applyResult,
    formValidationError,
    onApplyPreset,
  };
}
