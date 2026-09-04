import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  getCampaignWizardSession,
  listCampaignOnboardingTemplates,
  postCampaignWizardSession,
} from '@/api/campaigns_api';
import { ApiError } from '@/api/client';
import type {
  CampaignOnboardingTemplate,
  CampaignWizardCommitResult,
  CampaignWizardSession,
  CampaignWizardSessionRequest,
} from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  microQueryParamToUsdInput,
  usdInputToMicroQueryParam,
} from '@/domains/campaigns/list/campaign_list_format';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { cn } from '@/lib/utils';

type TemplateKey = NonNullable<CampaignWizardSessionRequest['template_key']>;
type WizardStepId = 'traffic_source' | 'integration_template' | 'flow_skeleton' | 'budget' | 'review';

const WIZARD_STEPS: { id: WizardStepId; label: string }[] = [
  { id: 'traffic_source', label: 'Traffic' },
  { id: 'integration_template', label: 'Integration' },
  { id: 'flow_skeleton', label: 'Flow' },
  { id: 'budget', label: 'Budget' },
  { id: 'review', label: 'Review' },
];

export type CampaignWizardPanelProps = {
  customerOptions: CustomerComboboxOption[];
  onCampaignCreated?: (campaignId: string) => void;
};

function panelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}

function formatTimestamp(iso: string | undefined): string {
  if (!iso) {
    return '-';
  }
  const parsed = Date.parse(iso);
  if (!Number.isFinite(parsed)) {
    return iso;
  }
  return new Date(parsed).toLocaleString();
}

function stepIndex(step: string | undefined): number {
  const index = WIZARD_STEPS.findIndex((row) => row.id === step);
  return index >= 0 ? index : 0;
}

function defaultTrafficDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    name: template?.title ?? '',
    traffic_template_id: 'default_rtb',
    click_query_params: JSON.stringify(template?.sample_macros ?? {}, null, 2),
  };
}

function defaultIntegrationDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    integration_schema: template?.integration_schema_refs?.[0] ?? '',
    affiliate_network: '',
    tracking_domain: '',
  };
}

function defaultFlowDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    flow_name: template?.default_flow?.flow_name ?? '',
    lander_name: template?.default_flow?.lander?.name ?? '',
    lander_url: template?.default_flow?.lander?.url ?? '',
    offer_name: template?.default_flow?.offer?.name ?? '',
    offer_url: template?.default_flow?.offer?.url ?? '',
  };
}

function defaultBudgetDraft() {
  return {
    budget_usd: '500.00',
    timezone: 'UTC',
    target_countries: 'US',
  };
}

export function CampaignWizardPanel({ customerOptions, onCampaignCreated }: CampaignWizardPanelProps) {
  const [searchParams] = useSearchParams();
  const { session: authSession } = useSession();
  const defaultCustomerId =
    searchParams.get('customer_id') ?? authSession?.default_customer_id ?? customerOptions[0]?.id ?? '';

  const templatesResource = useResource(listCampaignOnboardingTemplates, []);

  const [draftCustomerId, setDraftCustomerId] = useState(defaultCustomerId);
  const [draftTemplateKey, setDraftTemplateKey] = useState<TemplateKey | ''>('');
  const [sessionId, setSessionId] = useState('');
  const [session, setSession] = useState<CampaignWizardSession | undefined>();
  const [commitResult, setCommitResult] = useState<CampaignWizardCommitResult | undefined>();
  const [publishOnCommit, setPublishOnCommit] = useState(false);
  const [creating, setCreating] = useState(false);
  const [savingStep, setSavingStep] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [pollToken, setPollToken] = useState(0);

  const [trafficDraft, setTrafficDraft] = useState(defaultTrafficDraft(undefined));
  const [integrationDraft, setIntegrationDraft] = useState(defaultIntegrationDraft(undefined));
  const [flowDraft, setFlowDraft] = useState(defaultFlowDraft(undefined));
  const [budgetDraft, setBudgetDraft] = useState(defaultBudgetDraft());

  const templates = useMemo(() => templatesResource.data ?? [], [templatesResource.data]);

  useEffect(() => {
    if (templates.length === 0 || draftTemplateKey) {
      return;
    }
    setDraftTemplateKey(templates[0]?.key as TemplateKey);
  }, [draftTemplateKey, templates]);

  const selectedTemplate = useMemo(
    () => templates.find((item) => item.key === draftTemplateKey),
    [draftTemplateKey, templates],
  );

  useEffect(() => {
    if (!selectedTemplate || session) {
      return;
    }
    setTrafficDraft(defaultTrafficDraft(selectedTemplate));
    setIntegrationDraft(defaultIntegrationDraft(selectedTemplate));
    setFlowDraft(defaultFlowDraft(selectedTemplate));
  }, [selectedTemplate, session]);

  const sessionResource = useResource(
    (signal) => {
      if (!sessionId.trim()) {
        return Promise.resolve(undefined);
      }
      return getCampaignWizardSession(sessionId.trim(), signal);
    },
    [sessionId, pollToken],
  );

  const activeSession = sessionResource.data ?? session;

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
        setSessionId(result.session_id);
        setSession(result as CampaignWizardSession);
        setPollToken((value) => value + 1);
        toast.success('Wizard session created');
      }
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [draftCustomerId, draftTemplateKey]);

  const onSaveStep = useCallback(async () => {
    const id = sessionId.trim();
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
              Object.entries(parsed).map(([key, value]) => [key, String(value)]),
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
      setSession(result as CampaignWizardSession);
      setPollToken((value) => value + 1);
      toast.success('Step saved');
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingStep(false);
    }
  }, [
    activeSession,
    budgetDraft,
    flowDraft,
    integrationDraft,
    sessionId,
    trafficDraft,
  ]);

  const onCommitSession = useCallback(async () => {
    const id = sessionId.trim();
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
        idempotency_key: crypto.randomUUID(),
        publish: publishOnCommit,
      });
      setCommitResult(result as CampaignWizardCommitResult);
      setSession(undefined);
      setSessionId('');
      toast.success('Campaign created from wizard');
      if ('campaign' in result && result.campaign?.id) {
        onCampaignCreated?.(result.campaign.id);
      }
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCommitting(false);
    }
  }, [onCampaignCreated, publishOnCommit, sessionId]);

  const currentStep = (activeSession?.current_step ?? 'traffic_source') as WizardStepId;
  const completed = new Set(activeSession?.completed_steps ?? []);
  const activeStepIndex = stepIndex(currentStep);

  return (
    <div className="grid gap-6 pb-2">
      {templatesResource.error
        ? panelError(templatesResource.error, 'Could not load onboarding templates')
        : null}
      {sessionResource.error ? panelError(sessionResource.error, 'Could not load wizard session') : null}
      {actionError ? panelError(actionError, 'Wizard action failed') : null}

      {commitResult?.campaign ? (
        <section className="rounded-md border border-border bg-card p-3 shadow-sm grid gap-3">
          <h3 className="text-sm font-semibold text-foreground">Campaign created</h3>
          <p className="text-sm text-muted-foreground">
            {commitResult.campaign.name}{' '}
            <span className="font-mono text-xs">({commitResult.campaign.id})</span>
          </p>
          {commitResult.published ? (
            <p className="text-sm text-muted-foreground">Published after commit.</p>
          ) : null}
          {commitResult.publish_check && !commitResult.publish_check.valid ? (
            <ErrorBlock
              message="Publish gate blocked activation. Open the editor to resolve field errors."
              title="Publish check failed"
            />
          ) : null}
          <div className="flex flex-wrap gap-2">
            <Button asChild type="button">
              <Link to={`/campaigns/${commitResult.campaign.id}/edit`}>Open editor</Link>
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setCommitResult(undefined);
                setSession(undefined);
                setSessionId('');
              }}
            >
              Start another
            </Button>
          </div>
        </section>
      ) : null}

      {!activeSession ? (
        <section className="rounded-md border border-border bg-card p-3 grid gap-4">
          <div>
            <h3 className="text-sm font-semibold text-foreground">Setup</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Pick a customer and bundled onboarding template. The server stores a draft session for 24 hours.
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="wizard-customer">Customer</Label>
              <Select
                disabled={creating || customerOptions.length === 0}
                value={draftCustomerId || undefined}
                onValueChange={setDraftCustomerId}
              >
                <SelectTrigger id="wizard-customer">
                  <SelectValue placeholder="Select customer" />
                </SelectTrigger>
                <SelectContent>
                  {customerOptions.length === 0 ? (
                    <SelectItem disabled value="__none__">
                      No customers
                    </SelectItem>
                  ) : (
                    customerOptions.map((customer) => (
                      <SelectItem key={customer.id} value={customer.id}>
                        {customer.name}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="wizard-template">Template</Label>
              <Select
                disabled={creating || templates.length === 0}
                value={draftTemplateKey}
                onValueChange={(value) => setDraftTemplateKey(value as TemplateKey)}
              >
                <SelectTrigger id="wizard-template">
                  <SelectValue placeholder="Select template" />
                </SelectTrigger>
                <SelectContent>
                  {templates.map((template: CampaignOnboardingTemplate) => (
                    <SelectItem key={template.key} value={template.key ?? ''}>
                      {template.title ?? template.key}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {selectedTemplate ? (
            <div className="grid gap-2 rounded-md border border-border bg-muted p-3 text-sm">
              <p className="font-semibold text-foreground">{selectedTemplate.title}</p>
              <p className="text-muted-foreground">{selectedTemplate.description}</p>
              <p className="text-muted-foreground">
                Traffic family: <span className="text-foreground">{selectedTemplate.traffic_family}</span>
              </p>
              {selectedTemplate.integration_schema_refs?.length ? (
                <p className="text-muted-foreground">
                  Integration schemas:{' '}
                  <span className="font-mono text-xs text-foreground">
                    {selectedTemplate.integration_schema_refs.join(', ')}
                  </span>
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="flex flex-wrap gap-2">
            <Button
              disabled={creating || !draftCustomerId || !draftTemplateKey}
              loading={creating}
              type="button"
              onClick={onCreateSession}
            >
              Start wizard
            </Button>
          </div>
        </section>
      ) : (
        <>
          <section className="rounded-md border border-border bg-card p-3 grid gap-4">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="text-sm font-semibold text-foreground">Session</h3>
                <p className="mt-1 font-mono text-xs text-muted-foreground">{activeSession.session_id}</p>
              </div>
              <div className="text-right text-xs text-muted-foreground">
                <p>Expires {formatTimestamp(activeSession.expires_at)}</p>
                <p>Updated {formatTimestamp(activeSession.updated_at)}</p>
              </div>
            </div>

            <ol className="flex flex-wrap gap-2">
              {WIZARD_STEPS.map((step, index) => {
                const done = completed.has(step.id) || index < activeStepIndex;
                const active = step.id === currentStep;
                return (
                  <li
                    key={step.id}
                    className={cn(
                      'rounded-sm border px-2.5 py-1 text-xs font-medium',
                      done && 'border-blue-600/40 bg-blue-50 dark:bg-blue-950/40 text-blue-600 dark:text-blue-400',
                      active && !done && 'border-border bg-muted text-foreground',
                      !done && !active && 'border-border text-muted-foreground',
                    )}
                  >
                    {step.label}
                  </li>
                );
              })}
            </ol>
          </section>

          {currentStep === 'traffic_source' ? (
            <section className="rounded-md border border-border bg-card p-3 grid gap-4">
              <h3 className="text-sm font-semibold text-foreground">Traffic source</h3>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="text-sm font-medium text-foreground sm:col-span-2 grid gap-2">
                  <Label>Campaign name</Label>
                  <Input
                    value={trafficDraft.name}
                    onChange={(event) =>
                      setTrafficDraft((current) => ({ ...current, name: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Traffic template ID</Label>
                  <Input
                    value={trafficDraft.traffic_template_id}
                    onChange={(event) =>
                      setTrafficDraft((current) => ({
                        ...current,
                        traffic_template_id: event.target.value,
                      }))
                    }
                  />
                </div>
              </div>
              <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Click query params (JSON)</Label>
                  <Textarea className="min-h-28 font-mono text-xs"
                  value={trafficDraft.click_query_params}
                  onChange={(event) =>
                    setTrafficDraft((current) => ({
                      ...current,
                      click_query_params: event.target.value,
                    }))
                  }
                />
              </div>
              <div className="flex justify-end">
                <Button disabled={savingStep} loading={savingStep} type="button" onClick={onSaveStep}>
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'integration_template' ? (
            <section className="rounded-md border border-border bg-card p-3 grid gap-4">
              <h3 className="text-sm font-semibold text-foreground">Integration template</h3>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Integration schema</Label>
                  <Input
                    list="wizard-integration-schemas"
                    value={integrationDraft.integration_schema}
                    onChange={(event) =>
                      setIntegrationDraft((current) => ({
                        ...current,
                        integration_schema: event.target.value,
                      }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Affiliate network (optional)</Label>
                  <Input
                    value={integrationDraft.affiliate_network}
                    onChange={(event) =>
                      setIntegrationDraft((current) => ({
                        ...current,
                        affiliate_network: event.target.value,
                      }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground sm:col-span-2 grid gap-2">
                  <Label>Tracking domain (optional)</Label>
                  <Input
                    placeholder="track.example.com"
                    value={integrationDraft.tracking_domain}
                    onChange={(event) =>
                      setIntegrationDraft((current) => ({
                        ...current,
                        tracking_domain: event.target.value,
                      }))
                    }
                  />
                </div>
              </div>
              <datalist id="wizard-integration-schemas">
                {selectedTemplate?.integration_schema_refs?.map((ref) => (
                  <option key={ref} value={ref} />
                ))}
              </datalist>
              <div className="flex justify-end">
                <Button disabled={savingStep} loading={savingStep} type="button" onClick={onSaveStep}>
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'flow_skeleton' ? (
            <section className="rounded-md border border-border bg-card p-3 grid gap-4">
              <h3 className="text-sm font-semibold text-foreground">Flow skeleton</h3>
              <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Flow name</Label>
                  <Input
                  value={flowDraft.flow_name}
                  onChange={(event) =>
                    setFlowDraft((current) => ({ ...current, flow_name: event.target.value }))
                  }
                />
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Lander name</Label>
                  <Input
                    value={flowDraft.lander_name}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, lander_name: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Lander URL</Label>
                  <Input
                    value={flowDraft.lander_url}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, lander_url: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Offer name</Label>
                  <Input
                    value={flowDraft.offer_name}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, offer_name: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Offer URL</Label>
                  <Input
                    value={flowDraft.offer_url}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, offer_url: event.target.value }))
                    }
                  />
                </div>
              </div>
              <div className="flex justify-end">
                <Button disabled={savingStep} loading={savingStep} type="button" onClick={onSaveStep}>
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'budget' ? (
            <section className="rounded-md border border-border bg-card p-3 grid gap-4">
              <h3 className="text-sm font-semibold text-foreground">Budget</h3>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Budget limit ($)</Label>
                  <Input
                    inputMode="decimal"
                    value={budgetDraft.budget_usd}
                    onChange={(event) =>
                      setBudgetDraft((current) => ({ ...current, budget_usd: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground grid gap-2">
                  <Label>Timezone</Label>
                  <Input
                    value={budgetDraft.timezone}
                    onChange={(event) =>
                      setBudgetDraft((current) => ({ ...current, timezone: event.target.value }))
                    }
                  />
                </div>
                <div className="text-sm font-medium text-foreground sm:col-span-2 grid gap-2">
                  <Label>Target countries (comma-separated)</Label>
                  <Input
                    placeholder="US, CA, GB"
                    value={budgetDraft.target_countries}
                    onChange={(event) =>
                      setBudgetDraft((current) => ({
                        ...current,
                        target_countries: event.target.value,
                      }))
                    }
                  />
                </div>
              </div>
              <div className="flex justify-end">
                <Button disabled={savingStep} loading={savingStep} type="button" onClick={onSaveStep}>
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'review' ? (
            <section className="rounded-md border border-border bg-card p-3 grid gap-4">
              <h3 className="text-sm font-semibold text-foreground">Review</h3>
              {activeSession.review?.preview ? (
                <dl className="grid gap-2 text-sm sm:grid-cols-2">
                  <div>
                    <dt className="text-muted-foreground">Campaign name</dt>
                    <dd>{activeSession.review.preview.campaign_name ?? '-'}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Traffic template</dt>
                    <dd className="font-mono text-xs">
                      {activeSession.review.preview.traffic_template_id ?? '-'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Integration schema</dt>
                    <dd className="font-mono text-xs">
                      {activeSession.review.preview.integration_schema ?? '-'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Flow</dt>
                    <dd>{activeSession.review.preview.flow_name ?? '-'}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Budget</dt>
                    <dd className="tabular-nums">
                      ${microQueryParamToUsdInput(String(activeSession.review.preview.budget_limit_micro ?? ''))}
                    </dd>
                  </div>
                  <div className="sm:col-span-2">
                    <dt className="text-muted-foreground">Target URL</dt>
                    <dd className="break-all font-mono text-xs">
                      {activeSession.review.preview.target_url ?? '-'}
                    </dd>
                  </div>
                </dl>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Complete all steps to generate the commit preview.
                </p>
              )}
              {activeSession.review?.warning_slugs?.length ? (
                <div className="rounded-sm border border-border bg-muted p-3 text-sm text-muted-foreground">
                  Warnings: {activeSession.review.warning_slugs.join(', ')}
                </div>
              ) : null}

              <div className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={publishOnCommit}
                  id="wizard-publish-on-commit"
                  onCheckedChange={(checked) => setPublishOnCommit(checked === true)}
                />
                <Label className="font-normal" htmlFor="wizard-publish-on-commit">
                  Publish campaign after commit
                </Label>
              </div>

              <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
                <Button
                  disabled={committing || !activeSession.ready_to_commit}
                  loading={committing}
                  type="button"
                  onClick={onCommitSession}
                >
                  Create campaign
                </Button>
              </div>
            </section>
          ) : null}
        </>
      )}
    </div>
  );
}
