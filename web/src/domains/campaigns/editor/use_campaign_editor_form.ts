// L3 campaign editor form: form undefined until syncFormFromCampaign; onFieldChange is no-op when form not seeded.
import { useCallback, useState } from 'react';

import {
  campaignToFormState,
  type CampaignEditorFormState,
} from '@/domains/campaigns/editor/campaign_editor';
import type { Campaign } from '@/api/types';

export const CAMPAIGN_EDITOR_EMPTY_FORM: CampaignEditorFormState = {
  name: '',
  status: '',
  budget_limit: '',
  pacing_mode: '',
  flow_id: '',
  brand_id: '',
  ingress_param: '',
  ingress_scale: '',
  ingress_max_micro: '',
  ingress_policy: '',
  traffic_template_id: '',
  click_query_params_json: '{}',
  click_filter_tier: 'full',
  mobile_biometrics_click_enabled: false,
  decoy_lander_id: '',
};

export function useCampaignEditorForm() {
  const [form, setForm] = useState<CampaignEditorFormState | undefined>(undefined);

  const onFieldChange = useCallback(
    <K extends keyof CampaignEditorFormState>(field: K, value: CampaignEditorFormState[K]) => {
      setForm((prev) => {
        if (!prev) {
          return prev;
        }
        return { ...prev, [field]: value };
      });
    },
    []
  );

  const syncFormFromCampaign = useCallback((campaign: Campaign) => {
    setForm(campaignToFormState(campaign));
  }, []);

  return {
    form,
    setForm,
    onFieldChange,
    syncFormFromCampaign,
  };
}
