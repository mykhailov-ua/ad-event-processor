import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { api } from '../helpers/api_client.js';
import { createCampaign, patchCampaign } from '../helpers/campaign_admin_api.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { ParseDecimal } from '../helpers/money.js';
import { defaultClickTemplate } from '../helpers/tracking_link.js';
import { trafficGuideSummary } from '../helpers/integration_kit.js';
import { applyCampaignTemplates } from '../helpers/integration_api.js';
import { testPostbackConfig, type PostbackDryRunResult } from '../helpers/postback_api.js';
import { fetchSelfServeTemplates } from '../helpers/selfserve_api.js';
import {
  FIRST_CAMPAIGN_WIZARD_STEPS,
  buildWizardClickURL,
  bundledTrafficTemplateForSource,
  canLeaveFirstCampaignWizardStep,
  firstCampaignWizardStepLabel,
  prevFirstCampaignWizardStep,
  validateFirstCampaignBasics,
  type FirstCampaignWizardStep,
} from '../helpers/first_campaign_wizard_model.js';
import {
  TRAFFIC_SOURCE_TEMPLATES,
  templateParamMap,
  trafficSourceById,
} from '../models/traffic_source_templates.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button, ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { IntegrationCopyRow } from '../components/integration_copy_row.js';

type SelfServeTemplateRow = { id: string; name: string; budget_limit: string };

/**
 * Load platform click URL template and tracking domain for wizard links.
 */
async function loadWizardPlatformContext(): Promise<{
  clickTemplate: string;
  trackingDomain: string;
}> {
  const [platRes, docRes] = await Promise.all([
    to(
      api<{
        config?: { tracking_domain?: string };
        click_url_template?: string;
      }>('/api/v1/settings/platform')
    ),
    to(api<{ tracking_domain?: string; click_url_template?: string }>('/api/v1/ops/doctor')),
  ]);
  const plat = platRes[0]?.data;
  const doc = docRes[0]?.data;
  const trackingDomain = plat?.config?.tracking_domain ?? doc?.tracking_domain ?? '';
  const clickTemplate =
    doc?.click_url_template ||
    plat?.click_url_template ||
    defaultClickTemplate(trackingDomain);
  return { clickTemplate, trackingDomain };
}

export function FirstCampaignWizardPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryCustomerId = searchParams.get('customer_id')?.trim() ?? '';

  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');

  const [step, setStep] = useState<FirstCampaignWizardStep>('campaign');
  const [templates, setTemplates] = useState<SelfServeTemplateRow[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(true);
  const [templatesError, setTemplatesError] = useState<unknown>(null);

  const [customerId, setCustomerId] = useState(queryCustomerId);
  const [templateId, setTemplateId] = useState('');
  const [campaignName, setCampaignName] = useState('');
  const [budgetInput, setBudgetInput] = useState('');
  const [trafficSourceId, setTrafficSourceId] = useState('direct-custom');
  const [landerMode, setLanderMode] = useState<'external' | 'hosted' | 'skip'>('external');
  const [targetUrl, setTargetUrl] = useState('');

  const [campaignId, setCampaignId] = useState('');
  const [clickTemplate, setClickTemplate] = useState('');
  const [trackingDomain, setTrackingDomain] = useState('');
  const [platformLoading, setPlatformLoading] = useState(true);

  const [busy, setBusy] = useState(false);
  const [stepError, setStepError] = useState<string | null>(null);
  const [postbackResult, setPostbackResult] = useState<PostbackDryRunResult | null>(null);

  const reloadTemplates = useCallback(async (custId: string) => {
    setTemplatesLoading(true);
    setTemplatesError(null);
    const [rows, err] = await to(fetchSelfServeTemplates(custId || undefined));
    setTemplatesLoading(false);
    if (err) {
      setTemplatesError(err);
      return;
    }
    const items = rows ?? [];
    setTemplates(items);
    if (items[0]?.id) setTemplateId(items[0].id);
  }, []);

  useEffect(() => {
    void reloadTemplates(customerId);
  }, [customerId, reloadTemplates]);

  useEffect(() => {
    if (queryCustomerId && isCustomerUuid(queryCustomerId)) {
      setCustomerId(queryCustomerId);
    }
  }, [queryCustomerId]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const [ctx, err] = await to(loadWizardPlatformContext());
      if (cancelled) return;
      setPlatformLoading(false);
      if (err || !ctx) return;
      setClickTemplate(ctx.clickTemplate);
      setTrackingDomain(ctx.trackingDomain);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const clickURL = useMemo(() => {
    if (!campaignId) return '';
    return buildWizardClickURL(clickTemplate || trackingDomain, campaignId, trafficSourceId);
  }, [campaignId, clickTemplate, trackingDomain, trafficSourceId]);

  const trafficNotes = trafficSourceById(trafficSourceId)?.notes ?? '';

  const trafficMacroMap = useMemo(() => {
    const tpl = trafficSourceById(trafficSourceId);
    return tpl ? templateParamMap(tpl) : {};
  }, [trafficSourceId]);

  const createCampaignStep = async () => {
    const basicsErr = validateFirstCampaignBasics({
      customerId,
      templateId,
      name: campaignName,
      budgetInput,
    });
    if (basicsErr) {
      setStepError(basicsErr);
      return;
    }
    const body: Record<string, unknown> = {
      template_id: templateId,
      name: campaignName.trim(),
    };
    const budgetTrim = budgetInput.trim();
    if (budgetTrim) {
      try {
        body.budget_limit_micro = ParseDecimal(budgetTrim);
      } catch {
        setStepError('Budget must be a positive decimal');
        return;
      }
    }
    setBusy(true);
    setStepError(null);
    const [data, err] = await to(createCampaign(customerId.trim(), body));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setStepError(mapServiceError(err).message);
      return;
    }
    if (!data?.id) {
      setStepError('Campaign created but id missing in response');
      return;
    }
    setCampaignId(data.id);
    pushToastMessage({ title: 'Campaign created', message: data.id });
    setStep('traffic');
  };

  const trafficStepNext = async () => {
    if (!campaignId) {
      setStepError('Create the campaign first');
      return;
    }
    const bundled = bundledTrafficTemplateForSource(trafficSourceId);
    if (bundled) {
      setBusy(true);
      setStepError(null);
      const [, err] = await to(
        applyCampaignTemplates(campaignId, {
          traffic_source: bundled,
          tracking_domain: trackingDomain.trim() || undefined,
        })
      );
      setBusy(false);
      if (err) {
        if (err instanceof ConfirmCancelledError) return;
        setStepError(mapServiceError(err).message);
        return;
      }
      pushToastMessage({
        title: 'Integration template applied',
        message: bundled,
      });
    }
    setStep('lander');
  };

  const landerStepNext = async () => {
    if (!campaignId) {
      setStepError('Create the campaign first');
      return;
    }
    if (landerMode === 'external') {
      const landing = targetUrl.trim();
      if (!landing) {
        setStepError('Enter a landing URL or choose Skip / hosted lander');
        return;
      }
      setBusy(true);
      setStepError(null);
      const [, err] = await to(patchCampaign(campaignId, { target_url: landing }));
      setBusy(false);
      if (err) {
        if (err instanceof ConfirmCancelledError) return;
        setStepError(mapServiceError(err).message);
        return;
      }
    }
    setStep('test_click');
  };

  const runPostbackTest = async () => {
    if (!campaignId) return;
    setBusy(true);
    setStepError(null);
    setPostbackResult(null);
    const [result, err] = await to(testPostbackConfig(campaignId));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      setStepError(mapServiceError(err).message);
      return;
    }
    setPostbackResult(result ?? null);
  };

  const goNext = () => {
    setStepError(null);
    if (step === 'campaign') {
      void createCampaignStep();
      return;
    }
    if (step === 'traffic') {
      void trafficStepNext();
      return;
    }
    if (step === 'lander') {
      void landerStepNext();
      return;
    }
    if (step === 'test_click') {
      setStep('test_postback');
      return;
    }
    if (step === 'test_postback') {
      setStep('done');
      return;
    }
  };

  const goBack = () => {
    setStepError(null);
    const prev = prevFirstCampaignWizardStep(step);
    if (prev) setStep(prev);
  };

  if (!canWrite) {
    return (
      <section className="stack" data-testid="first-campaign-wizard">
        <p className="text-muted">Campaign write permission required.</p>
      </section>
    );
  }

  if (templatesError) {
    return (
      <section className="stack" data-testid="first-campaign-wizard">
        <ErrorBlock error={templatesError} fallbackTitle="Templates unavailable" />
      </section>
    );
  }

  const stepIndex = FIRST_CAMPAIGN_WIZARD_STEPS.indexOf(step);
  const isLastActionStep = step === 'test_postback';

  return (
    <section className="stack" data-testid="first-campaign-wizard">
      <Breadcrumbs
        items={[
          { label: 'Campaigns', href: '/campaigns' },
          { label: 'First campaign wizard' },
        ]}
      />
      <div className="page-header">
        <h1 className="page-header__title">First campaign wizard</h1>
        <p className="page-header__desc">{trafficGuideSummary()}</p>
      </div>

      <div className="filter-toolbar mb-4" aria-label="Wizard progress">
        <div className="filter-toolbar__chips" role="list">
          {FIRST_CAMPAIGN_WIZARD_STEPS.map((item, index) => {
            const active = item === step;
            const done = index < stepIndex;
            return (
              <span
                key={item}
                role="listitem"
                className={`chip${active ? ' chip--active' : ''}${done ? ' text-muted' : ''}`}
              >
                {firstCampaignWizardStepLabel(item)}
              </span>
            );
          })}
        </div>
      </div>

      <div className="section-card stack">
        {stepError ? <p className="text-danger text-sm">{stepError}</p> : null}

        {step === 'campaign' ? (
          <>
            <label className="form-field" htmlFor="fcw-customer">
              Customer UUID
              <input
                id="fcw-customer"
                className="form-input font-mono"
                value={customerId}
                data-testid="fcw-customer"
                onChange={(e) => setCustomerId(e.target.value.trim())}
              />
            </label>
            {templatesLoading ? <p className="text-muted">Loading templates...</p> : null}
            {!templatesLoading ? (
              <label className="form-field" htmlFor="fcw-template">
                Campaign template
                <select
                  id="fcw-template"
                  className="form-select"
                  value={templateId}
                  data-testid="fcw-template"
                  onChange={(e) => setTemplateId(e.target.value)}
                >
                  {templates.map((row) => (
                    <option key={row.id} value={row.id}>
                      {row.name} (budget {row.budget_limit})
                    </option>
                  ))}
                </select>
              </label>
            ) : null}
            <label className="form-field" htmlFor="fcw-name">
              Campaign name
              <input
                id="fcw-name"
                className="form-input"
                value={campaignName}
                data-testid="fcw-name"
                onChange={(e) => setCampaignName(e.target.value)}
              />
            </label>
            <label className="form-field" htmlFor="fcw-budget">
              Budget override USD (optional)
              <input
                id="fcw-budget"
                className="form-input"
                inputMode="decimal"
                value={budgetInput}
                data-testid="fcw-budget"
                onChange={(e) => setBudgetInput(e.target.value)}
              />
            </label>
          </>
        ) : null}

        {step === 'traffic' ? (
          <>
            <p className="text-muted text-sm">
              Pick the ad network you buy traffic from. Macros are filled into the click URL; bundled
              integration schemas apply when available.
            </p>
            <label className="form-field" htmlFor="fcw-traffic-source">
              Traffic source template
              <select
                id="fcw-traffic-source"
                className="form-select"
                value={trafficSourceId}
                data-testid="fcw-traffic-source"
                onChange={(e) => setTrafficSourceId(e.target.value)}
              >
                {TRAFFIC_SOURCE_TEMPLATES.map((tpl) => (
                  <option key={tpl.id} value={tpl.id}>
                    {tpl.name}
                  </option>
                ))}
              </select>
            </label>
            {trafficNotes ? <p className="text-muted text-sm">{trafficNotes}</p> : null}
            {bundledTrafficTemplateForSource(trafficSourceId) ? (
              <p className="text-muted text-sm">
                Bundled schema:{' '}
                <span className="font-mono">{bundledTrafficTemplateForSource(trafficSourceId)}</span>{' '}
                (applied on Next)
              </p>
            ) : (
              <p className="text-muted text-sm">
                No bundled schema for this source; copy the click URL manually on the next steps.
              </p>
            )}
            {campaignId && !platformLoading ? (
              <IntegrationCopyRow
                label="Click URL preview"
                value={clickURL}
                testId="fcw-click-url-preview"
              />
            ) : null}
            <details className="text-sm">
              <summary>Macro map</summary>
              <pre className="code-block">{JSON.stringify(trafficMacroMap, null, 2)}</pre>
            </details>
          </>
        ) : null}

        {step === 'lander' ? (
          <>
            <fieldset className="stack">
              <legend className="form-label">Lander destination</legend>
              <label className="form-field">
                <input
                  type="radio"
                  name="fcw-lander-mode"
                  checked={landerMode === 'external'}
                  onChange={() => setLanderMode('external')}
                />{' '}
                External URL (offer or pre-lander)
              </label>
              <label className="form-field">
                <input
                  type="radio"
                  name="fcw-lander-mode"
                  checked={landerMode === 'hosted'}
                  onChange={() => setLanderMode('hosted')}
                />{' '}
                Hosted ZIP lander (configure in Flows after wizard)
              </label>
              <label className="form-field">
                <input
                  type="radio"
                  name="fcw-lander-mode"
                  checked={landerMode === 'skip'}
                  onChange={() => setLanderMode('skip')}
                />{' '}
                Skip for now
              </label>
            </fieldset>
            {landerMode === 'external' ? (
              <label className="form-field" htmlFor="fcw-target-url">
                Landing URL
                <input
                  id="fcw-target-url"
                  className="form-input"
                  type="url"
                  placeholder="https://example.com/offer"
                  value={targetUrl}
                  data-testid="fcw-target-url"
                  onChange={(e) => setTargetUrl(e.target.value)}
                />
              </label>
            ) : null}
            {landerMode === 'hosted' ? (
              <p className="text-muted text-sm">
                After the wizard, open{' '}
                <Link to="/campaigns/flows">Campaign flows</Link> to upload a ZIP and attach the
                lander to a flow for this campaign.
              </p>
            ) : null}
          </>
        ) : null}

        {step === 'test_click' ? (
          <>
            <p className="text-muted text-sm">
              Paste the click URL into your traffic source or open it in a private window. A 302 to
              your lander means tracking is wired.
            </p>
            <IntegrationCopyRow label="Click URL" value={clickURL} testId="fcw-click-url" />
            <ButtonLink
              label="Open test click"
              variant="secondary"
              size="sm"
              href={clickURL}
              target="_blank"
              rel="noopener noreferrer"
            />
          </>
        ) : null}

        {step === 'test_postback' ? (
          <>
            <p className="text-muted text-sm">
              Dry-run dispatches the configured postback template for this campaign (no live
              affiliate call).
            </p>
            <Button
              label={busy ? 'Testing...' : 'Run postback test'}
              variant="secondary"
              size="sm"
              loading={busy}
              disabled={busy}
              data-testid="fcw-postback-test"
              onClick={() => void runPostbackTest()}
            />
            {postbackResult ? (
              <pre className="code-block" data-testid="fcw-postback-result">
                {JSON.stringify(postbackResult, null, 2)}
              </pre>
            ) : null}
          </>
        ) : null}

        {step === 'done' ? (
          <>
            <p className="text-muted text-sm">
              Campaign <span className="font-mono">{campaignId}</span> is ready. Tune filters, flows,
              and reporting on the detail page.
            </p>
            <div className="button-row">
              <ButtonLink
                label="Open campaign"
                variant="primary"
                href={`/campaigns/${campaignId}`}
              />
              <ButtonLink
                label="Cost Sync (optional)"
                variant="secondary"
                href="/integrations/cost-sync"
              />
              <ButtonLink
                label="CAPI and postbacks (optional)"
                variant="secondary"
                href="/integrations/postbacks"
              />
            </div>
          </>
        ) : null}

        {step !== 'done' ? (
          <div className="button-row">
            {prevFirstCampaignWizardStep(step) ? (
              <Button
                label="Back"
                variant="secondary"
                disabled={busy}
                onClick={goBack}
              />
            ) : (
              <Button
                label="Cancel"
                variant="secondary"
                disabled={busy}
                onClick={() => navigate('/campaigns')}
              />
            )}
            <Button
              label={
                busy
                  ? 'Working...'
                  : step === 'campaign'
                    ? 'Create and continue'
                    : isLastActionStep
                      ? 'Finish'
                      : 'Next'
              }
              variant="primary"
              loading={busy}
              disabled={busy || !canLeaveFirstCampaignWizardStep(step, campaignId)}
              data-testid="fcw-next"
              onClick={() => void goNext()}
            />
          </div>
        ) : (
          <Button label="Back to campaigns" variant="secondary" onClick={() => navigate('/campaigns')} />
        )}
      </div>
    </section>
  );
}
