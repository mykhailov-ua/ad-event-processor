import { Link } from 'react-router-dom';

import type { CampaignOnboardingTemplate } from '@/api/types';
import {
  campaignEditorActionsRowClass,
  campaignEditorFormColumnsClass,
  campaignEditorInsetPanelClass,
  campaignEditorSectionClass,
  campaignEditorWizardRootClass,
  campaignPanelError,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { ErrorBlock } from '@/shell/error_block';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT,
  CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS,
  CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MAX_HEIGHT_CLASS,
  CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MONO_CLASS,
} from '@/domains/campaigns/editor/campaign_click_query_limits';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  formatTimestamp,
  WIZARD_STEPS,
  type TemplateKey,
} from '@/domains/campaigns/editor/campaign_wizard_panel_shared';
import type { CampaignWizardPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_wizard_panel_workspace';
import { microQueryParamToUsdInput } from '@/domains/campaigns/list/campaign_list_format';
import { ADMIN_MONO_CLASS, ADMIN_SLUG_CLASS } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { SearchableFilterSelect } from '@/shell/searchable_filter_select';

export type CampaignWizardPanelProps = {
  workspace: CampaignWizardPanelWorkspace;
};

export function CampaignWizardPanel({ workspace }: CampaignWizardPanelProps) {
  const {
    load,
    customerOptions,
    templates,
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
  } = workspace;

  return (
    <div className={campaignEditorWizardRootClass} >
      {load.templatesError
        ? campaignPanelError(load.templatesError, 'Could not load onboarding templates')
        : null}
      {load.sessionError
        ? campaignPanelError(load.sessionError, 'Could not load wizard session')
        : null}
      {actionError ? campaignPanelError(actionError, 'Wizard action failed') : null}

      {commitResult?.campaign ? (
        <section className={campaignEditorSectionClass} >
          <h3 className={adminTypography.sectionTitle} >Campaign created</h3>
          <p className={adminTypography.bodyMuted} >
            {commitResult.campaign.name}{' '}
            <span className={cn(ADMIN_MONO_CLASS, adminTypography.captionPlain)} >({commitResult.campaign.id})</span>
          </p>
          {commitResult.published ? (
            <p className={adminTypography.bodyMuted} >Published after commit.</p>
          ) : null}
          {commitResult.publish_check && !commitResult.publish_check.valid ? (
            <ErrorBlock
              message="Publish gate blocked activation. Open the editor to resolve field errors."
              title="Publish check failed"
            />
          ) : null}
          <div className="flex flex-wrap gap-2" >
            <Button asChild type="button">
              <Link to={`/campaigns/${commitResult.campaign.id}/edit`}>Open editor</Link>
            </Button>
            <Button type="button" variant="secondary" onClick={onStartAnother}>
              Start another
            </Button>
          </div>
        </section>
      ) : null}

      {!activeSession && !load.templatesError ? (
        <section className={campaignEditorSectionClass} >
          <div className="grid gap-1" >
            <h3 className={adminTypography.sectionTitle} >Setup</h3>
            <p className={adminTypography.bodyMuted} >
              Pick a customer and bundled onboarding template. The server stores a draft session for
              24 hours.
            </p>
          </div>

          {load.templatesFetching ? (
            <p className={adminTypography.bodyMuted} >Loading templates...</p>
          ) : templates == null ? null : templates.length === 0 ? (
            <p>No onboarding templates configured.</p>
          ) : (
            <>
              <div className="grid gap-2" >
                <div>
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

                <div>
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
                <div>
                  <p className="text-muted-foreground" >{selectedTemplate.title}</p>
                  <p className="text-muted-foreground" >{selectedTemplate.description}</p>
                  <p>
                    Traffic family:{' '}
                    <span>
                      {selectedTemplate.traffic_family}
                    </span>
                  </p>
                  {selectedTemplate.integration_schema_refs?.length ? (
                    <p>
                      Integration schemas:{' '}
                      <span>
                        {selectedTemplate.integration_schema_refs.join(', ')}
                      </span>
                    </p>
                  ) : null}
                </div>
              ) : null}

              <div>
                <Button
                  disabled={creating || !draftCustomerId || !draftTemplateKey}
                  loading={creating}
                  type="button"
                  onClick={onCreateSession}
                >
                  Start wizard
                </Button>
              </div>
            </>
          )}
        </section>
      ) : activeSession ? (
        <>
          <section>
            <div className="grid gap-1" >
              <div>
                <h3>Session</h3>
                <p>
                  {activeSession.session_id}
                </p>
              </div>
              <div>
                <p>Expires {formatTimestamp(activeSession.expires_at)}</p>
                <p>Updated {formatTimestamp(activeSession.updated_at)}</p>
              </div>
            </div>

            <ol>
              {WIZARD_STEPS.map((step, index) => {
                const done = completed.has(step.id) || index < activeStepIndex;
                const active = step.id === currentStep;
                return (
                  <li
                    key={step.id}
                   
                  >
                    {step.label}
                  </li>
                );
              })}
            </ol>
          </section>

          {currentStep === 'traffic_source' ? (
            <section>
              <h3>Traffic source</h3>
              <div className={cn(adminTypography.label, 'sm:col-span-2 grid gap-2')} >
                <div>
                  <Label>Campaign name</Label>
                  <Input
                    value={trafficDraft.name}
                    onChange={(event) =>
                      setTrafficDraft((current) => ({ ...current, name: event.target.value }))
                    }
                  />
                </div>
                <div>
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
              <div>
                <Label>Click query params (JSON)</Label>
                <Textarea
                 
                  maxLength={CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS}
                  placeholder="{}"
                  showCount
                  value={trafficDraft.click_query_params}
                  onChange={(event) =>
                    setTrafficDraft((current) => ({
                      ...current,
                      click_query_params: event.target.value,
                    }))
                  }
                />
                <p>
                  {CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT}
                </p>
              </div>
              <div>
                <Button
                  disabled={savingStep}
                  loading={savingStep}
                  type="button"
                  onClick={onSaveStep}
                >
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'integration_template' ? (
            <section>
              <h3>Integration template</h3>
              <div className={cn(adminTypography.label, 'grid gap-2')} >
                <div>
                  <Label htmlFor="wizard-integration-schema">Integration schema</Label>
                  {(selectedTemplate?.integration_schema_refs?.length ?? 0) === 0 ? (
                    <Input
                      id="wizard-integration-schema"
                      value={integrationDraft.integration_schema}
                      onChange={(event) =>
                        setIntegrationDraft((current) => ({
                          ...current,
                          integration_schema: event.target.value,
                        }))
                      }
                    />
                  ) : (selectedTemplate?.integration_schema_refs?.length ?? 0) <= 6 ? (
                    <Select
                      value={integrationDraft.integration_schema || undefined}
                      onValueChange={(next) =>
                        setIntegrationDraft((current) => ({
                          ...current,
                          integration_schema: next,
                        }))
                      }
                    >
                      <SelectTrigger id="wizard-integration-schema">
                        <SelectValue placeholder="Select integration schema" />
                      </SelectTrigger>
                      <SelectContent>
                        {selectedTemplate?.integration_schema_refs?.map((ref) => (
                          <SelectItem key={ref} value={ref}>
                            {ref}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : (
                    <SearchableFilterSelect
                      allowFreeform
                      aria-label="Integration schema"
                      options={(selectedTemplate?.integration_schema_refs ?? []).map((ref) => ({
                        value: ref,
                        label: ref,
                      }))}
                      showSearch
                      triggerId="wizard-integration-schema"
                      value={integrationDraft.integration_schema}
                      onValueChange={(next) =>
                        setIntegrationDraft((current) => ({
                          ...current,
                          integration_schema: next,
                        }))
                      }
                    />
                  )}
                </div>
                <div>
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
                <div>
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
              <div>
                <Button
                  disabled={savingStep}
                  loading={savingStep}
                  type="button"
                  onClick={onSaveStep}
                >
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'flow_skeleton' ? (
            <section>
              <h3>Flow skeleton</h3>
              <p>
                Create a single-path flow here, or use the campaign editor paths panel for split
                tests.
              </p>
              <div className={cn(adminTypography.label, 'grid gap-2')} >
                <Label>Flow name</Label>
                <Input
                  value={flowDraft.flow_name}
                  onChange={(event) =>
                    setFlowDraft((current) => ({ ...current, flow_name: event.target.value }))
                  }
                />
              </div>
              <div className={campaignEditorFormColumnsClass} >
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Lander name</Label>
                  <Input
                    value={flowDraft.lander_name}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, lander_name: event.target.value }))
                    }
                  />
                </div>
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Lander URL</Label>
                  <Input
                    value={flowDraft.lander_url}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, lander_url: event.target.value }))
                    }
                  />
                </div>
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Offer name</Label>
                  <Input
                    value={flowDraft.offer_name}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, offer_name: event.target.value }))
                    }
                  />
                </div>
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Offer URL</Label>
                  <Input
                    value={flowDraft.offer_url}
                    onChange={(event) =>
                      setFlowDraft((current) => ({ ...current, offer_url: event.target.value }))
                    }
                  />
                </div>
              </div>
              <div className={campaignEditorActionsRowClass} >
                <Button
                  disabled={savingStep}
                  loading={savingStep}
                  type="button"
                  onClick={onSaveStep}
                >
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'budget' ? (
            <section className={campaignEditorSectionClass} >
              <h3 className={adminTypography.sectionTitle} >Budget</h3>
              <div className={campaignEditorFormColumnsClass} >
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Budget limit ($)</Label>
                  <Input
                    inputMode="decimal"
                    value={budgetDraft.budget_usd}
                    onChange={(event) =>
                      setBudgetDraft((current) => ({ ...current, budget_usd: event.target.value }))
                    }
                  />
                </div>
                <div className={cn(adminTypography.label, 'grid gap-2')} >
                  <Label>Timezone</Label>
                  <Input
                    value={budgetDraft.timezone}
                    onChange={(event) =>
                      setBudgetDraft((current) => ({ ...current, timezone: event.target.value }))
                    }
                  />
                </div>
                <div className={cn(adminTypography.label, 'sm:col-span-2 grid gap-2')} >
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
              <div className={campaignEditorActionsRowClass} >
                <Button
                  disabled={savingStep}
                  loading={savingStep}
                  type="button"
                  onClick={onSaveStep}
                >
                  Save step
                </Button>
              </div>
            </section>
          ) : null}

          {currentStep === 'review' ? (
            <section className={campaignEditorSectionClass} >
              <h3 className={adminTypography.sectionTitle} >Review</h3>
              {activeSession.review?.preview ? (
                <dl className={cn('grid gap-2 sm:grid-cols-2', adminTypography.body)} >
                  <div>
                    <dt className="text-muted-foreground" >Campaign name</dt>
                    <dd>{activeSession.review.preview.campaign_name ?? '-'}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground" >Traffic template</dt>
                    <dd className={ADMIN_SLUG_CLASS} >
                      {activeSession.review.preview.traffic_template_id ?? '-'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground" >Integration schema</dt>
                    <dd className={ADMIN_SLUG_CLASS} >
                      {activeSession.review.preview.integration_schema ?? '-'}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground" >Flow</dt>
                    <dd>{activeSession.review.preview.flow_name ?? '-'}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground" >Budget</dt>
                    <dd className="tabular-nums" >
                      $
                      {microQueryParamToUsdInput(
                        String(activeSession.review.preview.budget_limit_micro ?? '')
                      )}
                    </dd>
                  </div>
                  <div className="sm:col-span-2" >
                    <dt className="text-muted-foreground" >Target URL</dt>
                    <dd className={cn(ADMIN_MONO_CLASS, "break-all")} >
                      {activeSession.review.preview.target_url ?? '-'}
                    </dd>
                  </div>
                </dl>
              ) : (
                <p className={adminTypography.bodyMuted} >
                  Complete all steps to generate the commit preview.
                </p>
              )}
              {activeSession.review?.warning_slugs?.length ? (
                <div className={campaignEditorInsetPanelClass} >
                  Warnings: {activeSession.review.warning_slugs.join(', ')}
                </div>
              ) : null}

              <div className={cn(adminSpacing.flex.buttonGroup, adminTypography.body)} >
                <Checkbox
                  checked={publishOnCommit}
                  id="wizard-publish-on-commit"
                  onCheckedChange={(checked) => setPublishOnCommit(checked === true)}
                />
                <Label  className="font-normal" htmlFor="wizard-publish-on-commit">
                  Publish campaign after commit
                </Label>
              </div>

              <div className={campaignEditorActionsRowClass} >
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
      ) : null}
    </div>
  );
}
