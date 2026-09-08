// onboarding wizard: multi-step drafts + POST wizard session; URL step sync via useCampaignWizardPanelLoad.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import { postCampaignWizardSession } from '@/api/campaigns_api';
import type {
  CampaignOnboardingTemplate,
  CampaignWizardCommitResult,
  CampaignWizardSession,
} from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import {
  defaultBudgetDraft,
  defaultFlowDraft,
  defaultIntegrationDraft,
  defaultTrafficDraft,
  stepIndex,
  type TemplateKey,
  type WizardStepId,
} from '@/domains/campaigns/editor/campaign_wizard_panel_shared';
import {
  microQueryParamToUsdInput,
  usdInputToMicroQueryParam,
} from '@/domains/campaigns/list/campaign_list_format';
import { newRandomUuid } from '@/lib/uuid';
import { useCampaignWizardPanelLoad } from '@/domains/campaigns/editor/use_campaign_wizard_panel_load';
import { useSession } from '@/hooks/use_session';

type UseCampaignWizardPanelWorkspaceArgs = {
  enabled: boolean;
  customerOptions: CustomerComboboxOption[];
  onCampaignCreated?: (campaignId: string) => void;
};

export function useCampaignWizardPanelWorkspace({
  enabled,
  customerOptions,
  onCampaignCreated,
}: UseCampaignWizardPanelWorkspaceArgs) {
  const load = useCampaignWizardPanelLoad(enabled);
  const [searchParams] = useSearchParams();
  const { session: authSession } = useSession();
  const defaultCustomerId =
    searchParams.get('customer_id') ??
    authSession?.default_customer_id ??
    customerOptions[0]?.id ??
    '';

  const [draftCustomerId, setDraftCustomerId] = useState(defaultCustomerId);
  const [draftTemplateKey, setDraftTemplateKey] = useState<TemplateKey | ''>('');
  const [commitResult, setCommitResult] = useState<CampaignWizardCommitResult | undefined>();
  const [publishOnCommit, setPublishOnCommit] = useState(false);
  const [creating, setCreating] = useState(false);
  const [savingStep, setSavingStep] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();

  const [trafficDraft, setTrafficDraft] = useState(defaultTrafficDraft(undefined));
  const [integrationDraft, setIntegrationDraft] = useState(defaultIntegrationDraft(undefined));
  const [flowDraft, setFlowDraft] = useState(defaultFlowDraft(undefined));
  const [budgetDraft, setBudgetDraft] = useState(defaultBudgetDraft());

  const templates = load.templates;

  useEffect(() => {
    if (templates.length === 0 || draftTemplateKey) {
      return;
    }
    setDraftTemplateKey(templates[0]?.key as TemplateKey);
  }, [draftTemplateKey, templates]);

  useEffect(() => {
    if (enabled) {
      return;
    }
    setActionError(undefined);
    setCommitResult(undefined);
    setCreating(false);
    setSavingStep(false);
    setCommitting(false);
    setPublishOnCommit(false);
    setDraftTemplateKey('');
    setTrafficDraft(defaultTrafficDraft(undefined));
    setIntegrationDraft(defaultIntegrationDraft(undefined));
    setFlowDraft(defaultFlowDraft(undefined));
    setBudgetDraft(defaultBudgetDraft());
    load.resetSession();
  }, [enabled, load]);

  const selectedTemplate = useMemo(
    () => templates.find((item) => item.key === draftTemplateKey),
    [draftTemplateKey, templates]
  );

  useEffect(() => {
    if (!selectedTemplate || load.session) {
      return;
    }
    setTrafficDraft(defaultTrafficDraft(selectedTemplate));
    setIntegrationDraft(defaultIntegrationDraft(selectedTemplate));
    setFlowDraft(defaultFlowDraft(selectedTemplate));
  }, [load.session, selectedTemplate]);

  const activeSession = load.session;

  useEffect(() => {
    if (!activeSession) {
      return;
    }
    const steps = activeSession.steps;
    if (steps?.traffic_source) {
      setTrafficDraft({
        name: steps.traffic_source.name ?? '',
        traffic_template_id: steps.traffic_source.traffic_template_id ?? 'default_rtb',
        click_query_params: JSON.stringify(steps.traffic_source.click_query_params ?? {}, null, 2),
      });
    }
    if (steps?.integration_template) {
      setIntegrationDraft({
        integration_schema: steps.integration_template.integration_schema ?? '',
        affiliate_network: steps.integration_template.affiliate_network ?? '',
        tracking_domain: steps.integration_template.tracking_domain ?? '',
      });
    }
    if (steps?.flow_skeleton) {
      setFlowDraft({
        flow_name: steps.flow_skeleton.flow_name ?? '',
        lander_name: steps.flow_skeleton.lander?.name ?? '',
        lander_url: steps.flow_skeleton.lander?.url ?? '',
        offer_name: steps.flow_skeleton.offer?.name ?? '',
        offer_url: steps.flow_skeleton.offer?.url ?? '',
      });
    }
    if (steps?.budget) {
      setBudgetDraft({
        budget_usd: microQueryParamToUsdInput(String(steps.budget.budget_limit_micro ?? '')),
        timezone: steps.budget.timezone ?? 'UTC',
        target_countries: (steps.budget.target_countries ?? []).join(', '),
      });
    }
  }, [activeSession]);

  const onCreateSession = useCallback(async () => {
    const customerId = draftCustomerId.trim();
    if (!customerId) {
      setActionError(new Error('Customer is required.'));
      return;
    }
    if (!draftTemplateKey) {
      setActionError(new Error('Template is required.'));
      return;
    }
    setCreating(true);
    setActionError(undefined);
    setCommitResult(undefined);
    try {
      const result = await postCampaignWizardSession({
        action: 'create',
        customer_id: customerId,
        template_key: draftTemplateKey,
      });
      if ('session_id' in result && result.session_id) {
        load.onSessionCreated(result.session_id, result as CampaignWizardSession);
        toast.success('Wizard session created');
      }
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftCustomerId, draftTemplateKey, load]);

  const onSaveStep = useCallback(async () => {
    const id = load.sessionId.trim();
    if (!id || !activeSession) {
      return;
    }
    const step = activeSession.current_step as WizardStepId;
    if (step === 'review') {
      return;
    }

    let payload: Record<string, unknown>;
    if (step === 'traffic_source') {
      let clickQueryParams: Record<string, string> = {};
      if (trafficDraft.click_query_params.trim()) {
        try {
          const parsed = JSON.parse(trafficDraft.click_query_params) as unknown;
          if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
            clickQueryParams = Object.fromEntries(
              Object.entries(parsed).map(([key, value]) => [key, String(value)])
            );
          }
        } catch {
          setActionError(new Error('Click query params must be valid JSON.'));
          return;
        }
      }
      payload = {
        name: trafficDraft.name.trim(),
        traffic_template_id: trafficDraft.traffic_template_id.trim(),
        click_query_params: clickQueryParams,
      };
    } else if (step === 'integration_template') {
      payload = {
        integration_schema: integrationDraft.integration_schema.trim(),
        affiliate_network: integrationDraft.affiliate_network.trim() || undefined,
        tracking_domain: integrationDraft.tracking_domain.trim() || undefined,
      };
    } else if (step === 'flow_skeleton') {
      payload = {
        flow_name: flowDraft.flow_name.trim(),
        lander: { name: flowDraft.lander_name.trim(), url: flowDraft.lander_url.trim() },
        offer: { name: flowDraft.offer_name.trim(), url: flowDraft.offer_url.trim() },
      };
    } else {
      const budgetMicro = usdInputToMicroQueryParam(budgetDraft.budget_usd);
      if (budgetMicro == null || budgetMicro <= 0) {
        setActionError(new Error('Budget must be a positive USD amount.'));
        return;
      }
      payload = {
        budget_limit_micro: budgetMicro,
        timezone: budgetDraft.timezone.trim() || 'UTC',
        target_countries: budgetDraft.target_countries
          .split(',')
          .map((value) => value.trim())
          .filter(Boolean),
      };
    }

    setSavingStep(true);
    setActionError(undefined);
    try {
      const result = await postCampaignWizardSession({
        action: 'update',
        session_id: id,
        step,
        payload,
      });
      load.onSessionUpdated(result as CampaignWizardSession);
      toast.success('Step saved');
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingStep(false);
    }
  }, [activeSession, budgetDraft, flowDraft, integrationDraft, load, trafficDraft]);

  const onCommitSession = useCallback(async () => {
    const id = load.sessionId.trim();
    if (!id) {
      setActionError(new Error('Session is required to commit.'));
      return;
    }
    setCommitting(true);
    setActionError(undefined);
    try {
      const result = await postCampaignWizardSession({
        action: 'commit',
        session_id: id,
        idempotency_key: newRandomUuid(),
        publish: publishOnCommit,
      });
      setCommitResult(result as CampaignWizardCommitResult);
      load.onSessionCommitted();
      toast.success('Campaign created from wizard');
      if ('campaign' in result && result.campaign?.id) {
        onCampaignCreated?.(result.campaign.id);
      }
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCommitting(false);
    }
  }, [load, onCampaignCreated, publishOnCommit]);

  const onStartAnother = useCallback(() => {
    setCommitResult(undefined);
    load.onStartAnother();
  }, [load]);

  const currentStep = (activeSession?.current_step ?? 'traffic_source') as WizardStepId;
  const completed = new Set(activeSession?.completed_steps ?? []);
  const activeStepIndex = stepIndex(currentStep);

  return {
    load,
    customerOptions,
    templates: templates as CampaignOnboardingTemplate[],
    selectedTemplate,
    draftCustomerId,
    setDraftCustomerId,
    draftTemplateKey,
    setDraftTemplateKey,
    commitResult,
    publishOnCommit,
    setPublishOnCommit,
    creating,
    savingStep,
    committing,
    actionError,
    trafficDraft,
    setTrafficDraft,
    integrationDraft,
    setIntegrationDraft,
    flowDraft,
    setFlowDraft,
    budgetDraft,
    setBudgetDraft,
    activeSession,
    currentStep,
    completed,
    activeStepIndex,
    onCreateSession,
    onSaveStep,
    onCommitSession,
    onStartAnother,
  };
}

export type CampaignWizardPanelWorkspace = ReturnType<typeof useCampaignWizardPanelWorkspace>;
