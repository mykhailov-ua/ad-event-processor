import { Link } from 'react-router-dom';
import { useEffect, useState, type ReactNode } from 'react';

import type { FlowPath } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import { ApiError } from '@/api/client';
import type {
  CampaignDiffResponse,
  CloneCampaignOptions,
  CloneCampaignPreview,
} from '@/api/campaigns_api';
import type {
  Campaign,
  CampaignPublishBlockedError,
  CampaignPublishCheck,
  CampaignValidateResponse,
  IngressCostConfig,
  PatchCampaignRequest,
} from '@/api/types';
import type { MacroPreviewResponse } from '@/api/campaigns_api';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { CampaignEditorBinomShell } from '@/domains/campaigns/editor/campaign_editor_binom_shell';
import { CampaignEditorTools } from '@/domains/campaigns/editor/campaign_editor_tools';

export type CampaignEditorFormState = {
  name: string;
  status: string;
  budget_limit: string;
  pacing_mode: string;
  flow_id: string;
  brand_id: string;
  ingress_param: string;
  ingress_scale: string;
  ingress_max_micro: string;
  ingress_policy: string;
  traffic_template_id: string;
  click_query_params_json: string;
};

export type BuildCampaignPatchResult =
  | { ok: true; body: PatchCampaignRequest }
  | { ok: false; error: string };

export type MacroPreviewFormState = {
  sub1: string;
  country: string;
  click_id: string;
};

export type CampaignEditorProps = {
  campaign: Campaign | undefined;
  flowPaths?: FlowPath[];
  form: CampaignEditorFormState;
  fetching: boolean;
  saving: boolean;
  loadError: Error | undefined;
  saveError: Error | undefined;
  hasSnapshot: boolean;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K],
  ) => void;
  onSave: () => void;
  onSaveAndClose?: () => void;
  onOpenClone?: () => void;
  clickUrl?: string;
  checking: boolean;
  validating: boolean;
  publishing: boolean;
  forcePublish: boolean;
  publishCheck: CampaignPublishCheck | undefined;
  validateResult: CampaignValidateResponse | undefined;
  publishBlocked: CampaignPublishBlockedError | undefined;
  publishSuccess: boolean;
  publishCheckError: Error | undefined;
  validateError: Error | undefined;
  publishError: Error | undefined;
  onForcePublishChange: (force: boolean) => void;
  onCheckPublish: () => void;
  onValidateChanges: () => void;
  onPublish: () => void;
  macroPreviewForm: MacroPreviewFormState;
  onMacroPreviewFieldChange: <K extends keyof MacroPreviewFormState>(
    field: K,
    value: MacroPreviewFormState[K],
  ) => void;
  macroPreviewing: boolean;
  macroPreviewResult: MacroPreviewResponse | undefined;
  macroPreviewError: Error | undefined;
  onMacroPreview: () => void;
  cloneNameSuffix: string;
  onCloneNameSuffixChange: (value: string) => void;
  cloneOptions: CloneCampaignOptions;
  onCloneOptionChange: (field: keyof CloneCampaignOptions, value: boolean) => void;
  clonePreviewing: boolean;
  clonePreview: CloneCampaignPreview | undefined;
  clonePreviewError: Error | undefined;
  onClonePreview: () => void;
  cloning: boolean;
  cloneSuccess: boolean;
  clonedCampaignId: string | undefined;
  cloneError: Error | undefined;
  onCloneExecute: () => void;
  diffAgainstId: string;
  onDiffAgainstIdChange: (value: string) => void;
  comparingDiff: boolean;
  diffResult: CampaignDiffResponse | undefined;
  diffError: Error | undefined;
  onCompareDiff: () => void;
  draftOwnerUserId: string;
  onDraftOwnerUserIdChange: (value: string) => void;
  transferringOwner: boolean;
  ownerError: Error | undefined;
  ownerSuccess: boolean;
  onTransferOwner: () => void;
  exporting: boolean;
  exportError: Error | undefined;
  onExportCampaign: () => void;
};

type CampaignDisplayFields = Campaign & {
  budget_limit_display?: string;
  current_spend_display?: string;
  daily_budget_display?: string;
  status_label?: string;
};

function formatReadonly(value: string | undefined): string {
  if (value == null || value === '') {
    return '-';
  }
  return value;
}

function ingressFromForm(form: CampaignEditorFormState): IngressCostConfig | undefined {
  const param = form.ingress_param.trim();
  if (param === '') {
    return undefined;
  }

  const config: IngressCostConfig = { param };
  const scale = form.ingress_scale.trim();
  if (scale !== '') {
    config.scale = scale;
  }
  const maxMicroText = form.ingress_max_micro.trim();
  if (maxMicroText !== '') {
    const parsed = Number(maxMicroText);
    if (!Number.isNaN(parsed)) {
      config.max_micro = parsed;
    }
  }
  const policy = form.ingress_policy.trim();
  if (policy !== '') {
    config.policy = policy;
  }
  return config;
}

function ingressConfigsEqual(
  left: IngressCostConfig | undefined,
  right: IngressCostConfig | undefined,
): boolean {
  return (
    (left?.param ?? '') === (right?.param ?? '') &&
    (left?.scale ?? '') === (right?.scale ?? '') &&
    String(left?.max_micro ?? '') === String(right?.max_micro ?? '') &&
    (left?.policy ?? '') === (right?.policy ?? '')
  );
}

function clickQueryParamsCanonicalJson(params: Record<string, string> | undefined): string {
  return JSON.stringify(params ?? {}, null, 2);
}

export function parseClickQueryParamsJson(
  json: string,
): { ok: true; value: Record<string, string> } | { ok: false; error: string } {
  const trimmed = json.trim();
  if (trimmed === '') {
    return { ok: true, value: {} };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, error: 'Click query params must be valid JSON.' };
  }

  if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return { ok: false, error: 'Click query params must be a JSON object.' };
  }

  const value: Record<string, string> = {};
  for (const [key, entry] of Object.entries(parsed as Record<string, unknown>)) {
    if (typeof entry !== 'string') {
      return {
        ok: false,
        error: `Click query params value for "${key}" must be a string.`,
      };
    }
    value[key] = entry;
  }
  return { ok: true, value };
}

function clickQueryParamsEqual(
  left: Record<string, string> | undefined,
  right: Record<string, string>,
): boolean {
  const leftObj = left ?? {};
  const leftKeys = Object.keys(leftObj).sort();
  const rightKeys = Object.keys(right).sort();
  if (leftKeys.length !== rightKeys.length) {
    return false;
  }
  return leftKeys.every((key, index) => key === rightKeys[index] && leftObj[key] === right[key]);
}

export function campaignToFormState(campaign: Campaign): CampaignEditorFormState {
  const ingress = campaign.ingress_cost_config;
  return {
    name: campaign.name,
    status: campaign.status,
    budget_limit: campaign.budget_limit,
    pacing_mode: campaign.pacing_mode,
    flow_id: campaign.flow_id ?? '',
    brand_id: campaign.brand_id ?? '',
    ingress_param: ingress?.param ?? '',
    ingress_scale: ingress?.scale ?? '',
    ingress_max_micro: ingress?.max_micro != null ? String(ingress.max_micro) : '',
    ingress_policy: ingress?.policy ?? '',
    traffic_template_id: campaign.traffic_template_id ?? '',
    click_query_params_json: clickQueryParamsCanonicalJson(campaign.click_query_params),
  };
}

export function buildCampaignPatchBody(
  original: Campaign,
  form: CampaignEditorFormState,
): BuildCampaignPatchResult {
  const body: PatchCampaignRequest = {};

  if (form.name !== original.name) {
    body.name = form.name;
  }
  if (form.status !== original.status) {
    body.status = form.status;
  }
  if (form.budget_limit !== original.budget_limit) {
    body.budget_limit = form.budget_limit;
  }
  if (form.pacing_mode !== original.pacing_mode) {
    body.pacing_mode = form.pacing_mode;
  }
  if (form.flow_id !== (original.flow_id ?? '')) {
    body.flow_id = form.flow_id;
  }
  if (form.brand_id !== (original.brand_id ?? '')) {
    body.brand_id = form.brand_id;
  }

  const nextIngress = ingressFromForm(form);
  if (!ingressConfigsEqual(nextIngress, original.ingress_cost_config)) {
    body.ingress_cost_config = nextIngress;
  }

  if (form.traffic_template_id !== (original.traffic_template_id ?? '')) {
    body.traffic_template_id = form.traffic_template_id;
  }

  const originalClickQueryJson = clickQueryParamsCanonicalJson(original.click_query_params);
  if (form.click_query_params_json !== originalClickQueryJson) {
    const parsed = parseClickQueryParamsJson(form.click_query_params_json);
    if (!parsed.ok) {
      return { ok: false, error: parsed.error };
    }
    if (!clickQueryParamsEqual(original.click_query_params, parsed.value)) {
      body.click_query_params = parsed.value;
    }
  }

  return { ok: true, body };
}

function fieldErrorEntries(
  fieldErrors: Record<string, string> | undefined,
): [string, string][] {
  if (!fieldErrors) {
    return [];
  }
  return Object.entries(fieldErrors);
}

function FieldErrorsPanel({
  title,
  fieldErrors,
}: {
  title: string;
  fieldErrors: Record<string, string> | undefined;
}) {
  const entries = fieldErrorEntries(fieldErrors);
  if (entries.length === 0) {
    return null;
  }

  return (
    <div className="grid gap-2">
      <p className="text-sm font-medium">{title}</p>
      <ul className="list-inside list-disc text-sm text-muted-foreground">
        {entries.map(([field, message]) => (
          <li key={field}>
            <span className="font-mono text-xs">{field}</span>: {message}
          </li>
        ))}
      </ul>
      <pre className="overflow-x-auto rounded-md bg-muted p-2 text-xs">
        {JSON.stringify(fieldErrors, null, 2)}
      </pre>
    </div>
  );
}

function ValidityBadge({ valid, validLabel, invalidLabel }: {
  valid: boolean;
  validLabel: string;
  invalidLabel: string;
}) {
  return (
    <Badge variant={valid ? 'secondary' : 'destructive'}>
      {valid ? validLabel : invalidLabel}
    </Badge>
  );
}

const CLONE_OPTION_FIELDS: {
  field: keyof CloneCampaignOptions;
  label: string;
  description: string;
}[] = [
  {
    field: 'include_flow',
    label: 'Include flow',
    description: 'Copy flow routing from the source campaign.',
  },
  {
    field: 'include_postbacks',
    label: 'Include postbacks',
    description: 'Copy postback and integration URLs.',
  },
  {
    field: 'include_fraud',
    label: 'Include fraud settings',
    description: 'Copy fraud presets and overrides.',
  },
  {
    field: 'include_placement_blocks',
    label: 'Include placement blocks',
    description: 'Copy blocked placement rules.',
  },
  {
    field: 'reset_spend',
    label: 'Reset spend',
    description: 'Start the clone with zero spend counters.',
  },
];

function diffSeverityVariant(
  severity: string,
): 'secondary' | 'destructive' | 'outline' {
  if (severity === 'remove') {
    return 'destructive';
  }
  if (severity === 'add') {
    return 'secondary';
  }
  return 'outline';
}

function StringList({ title, items }: { title: string; items: string[] | undefined }) {
  if (!items || items.length === 0) {
    return null;
  }

  return (
    <div className="grid gap-1">
      <p className="text-sm font-medium">{title}</p>
      <ul className="list-inside list-disc text-sm text-muted-foreground">
        {items.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function EditorStatusBanners({
  saveError,
  publishCheckError,
  validateError,
  publishError,
}: {
  saveError: Error | undefined;
  publishCheckError: Error | undefined;
  validateError: Error | undefined;
  publishError: Error | undefined;
}): ReactNode {
  const blocks: ReactNode[] = [];
  if (saveError) {
    blocks.push(
      saveError instanceof ApiError && saveError.status === 501 ? (
        <StubBanner key="save" title="Save not available" message={saveError.message} />
      ) : (
        <ErrorBlock key="save" title="Could not save campaign" message={saveError.message} />
      ),
    );
  }
  if (publishCheckError) {
    blocks.push(
      <ErrorBlock
        key="publish-check"
        title="Could not check publish gate"
        message={publishCheckError.message}
      />,
    );
  }
  if (validateError) {
    blocks.push(
      <ErrorBlock key="validate" title="Could not validate changes" message={validateError.message} />,
    );
  }
  if (publishError) {
    blocks.push(
      <ErrorBlock key="publish" title="Could not publish campaign" message={publishError.message} />,
    );
  }
  if (blocks.length === 0) {
    return null;
  }
  return <div className="admin-stack">{blocks}</div>;
}

export function CampaignEditor({
  campaign,
  flowPaths,
  form,
  fetching,
  saving,
  loadError,
  saveError,
  hasSnapshot,
  onFieldChange,
  onSave,
  onSaveAndClose,
  onOpenClone,
  clickUrl,
  checking,
  validating,
  publishing,
  forcePublish,
  publishCheck,
  validateResult,
  publishBlocked,
  publishSuccess,
  publishCheckError,
  validateError,
  publishError,
  onForcePublishChange,
  onCheckPublish,
  onValidateChanges,
  onPublish,
  macroPreviewForm,
  onMacroPreviewFieldChange,
  macroPreviewing,
  macroPreviewResult,
  macroPreviewError,
  onMacroPreview,
  cloneNameSuffix,
  onCloneNameSuffixChange,
  cloneOptions,
  onCloneOptionChange,
  clonePreviewing,
  clonePreview,
  clonePreviewError,
  onClonePreview,
  cloning,
  cloneSuccess,
  clonedCampaignId,
  cloneError,
  onCloneExecute,
  diffAgainstId,
  onDiffAgainstIdChange,
  comparingDiff,
  diffResult,
  diffError,
  onCompareDiff,
  draftOwnerUserId,
  onDraftOwnerUserIdChange,
  transferringOwner,
  ownerError,
  ownerSuccess,
  onTransferOwner,
  exporting,
  exportError,
  onExportCampaign,
}: CampaignEditorProps) {
  const [cloneOpen, setCloneOpen] = useState(false);

  useEffect(() => {
    if (cloneSuccess) {
      setCloneOpen(false);
    }
  }, [cloneSuccess]);

  if (fetching && !hasSnapshot && !loadError) {
    return <PageSkeleton />;
  }

  if (loadError && !hasSnapshot) {
    if (loadError instanceof ApiError && loadError.status === 501) {
      return (
        <StubBanner
          title="Campaign editor unavailable"
          message={loadError.message}
        />
      );
    }
    return <ErrorBlock title="Could not load campaign" message={loadError.message} />;
  }

  if (!campaign) {
    return <ErrorBlock title="Campaign not found" message="No campaign data returned." />;
  }

  const displayCampaign = campaign as CampaignDisplayFields;
  const statusLabel = formatCampaignStatusLabel(
    displayCampaign.status,
    displayCampaign.status_label,
  );
  const gateBusy = checking || validating || publishing;

  return (
    <>
      <CampaignEditorBinomShell
        campaignId={campaign.id}
        campaignName={campaign.name}
        clickUrl={clickUrl ?? macroPreviewResult?.resolved_click_url}
        flowPaths={flowPaths}
        form={form}
        saving={saving}
        statusBanner={
          <EditorStatusBanners
            publishCheckError={publishCheckError}
            publishError={publishError}
            saveError={saveError}
            validateError={validateError}
          />
        }
        advancedPanel={
          <div className="admin-stack">
            <p className="admin-muted">
              Status: {statusLabel}
              {checking ? '  /  Checking publish...' : ''}
              {publishCheck && !checking ? (
                publishCheck.valid ? '  /  Publish ready' : '  /  Publish blocked'
              ) : (
                ''
              )}
            </p>
            <Card>
        <CardHeader>
          <CardTitle className="text-base">Routing & ingress</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="campaign-flow-id">Flow ID</Label>
              <Input
                id="campaign-flow-id"
                value={form.flow_id}
                disabled={saving}
                onChange={(event) => onFieldChange('flow_id', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="campaign-brand-id">Brand ID</Label>
              <Input
                id="campaign-brand-id"
                value={form.brand_id}
                disabled={saving}
                onChange={(event) => onFieldChange('brand_id', event.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-param">Ingress cost param</Label>
              <Input
                id="campaign-ingress-param"
                value={form.ingress_param}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_param', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-scale">Ingress cost scale</Label>
              <Input
                id="campaign-ingress-scale"
                value={form.ingress_scale}
                disabled={saving}
                placeholder="decimal or micro"
                onChange={(event) => onFieldChange('ingress_scale', event.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-max-micro">Ingress max micro</Label>
              <Input
                id="campaign-ingress-max-micro"
                value={form.ingress_max_micro}
                disabled={saving}
                inputMode="numeric"
                onChange={(event) => onFieldChange('ingress_max_micro', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-policy">Ingress policy</Label>
              <Input
                id="campaign-ingress-policy"
                value={form.ingress_policy}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_policy', event.target.value)}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Integrations</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="campaign-traffic-template-id">Traffic template ID</Label>
            <Input
              id="campaign-traffic-template-id"
              value={form.traffic_template_id}
              disabled={saving}
              placeholder="meta-facebook"
              onChange={(event) => onFieldChange('traffic_template_id', event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Integration click URL preset template (for example meta-facebook).
            </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="campaign-click-query-params">Click query params (JSON)</Label>
            <textarea
              id="campaign-click-query-params"
              className="min-h-32 w-full rounded-xl border border-border/50 bg-muted/40 px-3 py-2 font-mono text-sm"
              value={form.click_query_params_json}
              disabled={saving}
              placeholder={'{\n  "sub2": "{{campaign.id}}"\n}'}
              onChange={(event) => onFieldChange('click_query_params_json', event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Query param macros for the click URL preset (sub1..sub30, ad_campaign_id, click ids).
              Values must be strings.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Macro preview</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="grid gap-2">
              <Label htmlFor="macro-preview-sub1">sub1</Label>
              <Input
                id="macro-preview-sub1"
                value={macroPreviewForm.sub1}
                disabled={macroPreviewing || fetching}
                onChange={(event) => onMacroPreviewFieldChange('sub1', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="macro-preview-country">country</Label>
              <Input
                id="macro-preview-country"
                value={macroPreviewForm.country}
                disabled={macroPreviewing || fetching}
                onChange={(event) => onMacroPreviewFieldChange('country', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="macro-preview-click-id">click_id</Label>
              <Input
                id="macro-preview-click-id"
                value={macroPreviewForm.click_id}
                disabled={macroPreviewing || fetching}
                onChange={(event) => onMacroPreviewFieldChange('click_id', event.target.value)}
              />
            </div>
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="secondary"
              disabled={macroPreviewing || fetching}
              onClick={onMacroPreview}
            >
              {macroPreviewing ? 'Previewing...' : 'Preview'}
            </Button>
          </div>

          {macroPreviewError ? (
            macroPreviewError instanceof ApiError && macroPreviewError.status === 501 ? (
              <StubBanner title="Macro preview unavailable" message={macroPreviewError.message} />
            ) : (
              <ErrorBlock title="Could not preview macros" message={macroPreviewError.message} />
            )
          ) : null}

          {macroPreviewResult ? (
            <div className="admin-panel grid gap-3 p-3">
              <div className="grid gap-2">
                <p className="text-sm font-medium">Resolved click URL</p>
                <p className="break-all font-mono text-xs text-muted-foreground">
                  {formatReadonly(macroPreviewResult.resolved_click_url)}
                </p>
              </div>
              {macroPreviewResult.resolved_postback_url ? (
                <div className="grid gap-2">
                  <p className="text-sm font-medium">Resolved postback URL</p>
                  <p className="break-all font-mono text-xs text-muted-foreground">
                    {macroPreviewResult.resolved_postback_url}
                  </p>
                </div>
              ) : null}
              <StringList title="Warnings" items={macroPreviewResult.warnings} />
              <StringList title="Unresolved macros" items={macroPreviewResult.unresolved_macros} />
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Publish gate</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="secondary"
              disabled={gateBusy || fetching}
              onClick={onCheckPublish}
            >
              {checking ? 'Checking...' : 'Check publish'}
            </Button>
            <Button
              type="button"
              variant="secondary"
              disabled={gateBusy || fetching || saving}
              onClick={onValidateChanges}
            >
              {validating ? 'Validating...' : 'Validate changes'}
            </Button>
            <Button
              type="button"
              disabled={gateBusy || fetching || saving}
              onClick={onPublish}
            >
              {publishing ? 'Publishing...' : 'Publish'}
            </Button>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              checked={forcePublish}
              disabled={gateBusy || fetching}
              id="campaign-force-publish"
              onCheckedChange={(checked) => onForcePublishChange(checked === true)}
            />
            <Label htmlFor="campaign-force-publish">Force publish</Label>
          </div>

          {publishCheckError ? (
            publishCheckError instanceof ApiError && publishCheckError.status === 501 ? (
              <StubBanner title="Publish check unavailable" message={publishCheckError.message} />
            ) : (
              <ErrorBlock title="Could not check publish gate" message={publishCheckError.message} />
            )
          ) : null}

          {validateError ? (
            validateError instanceof ApiError && validateError.status === 501 ? (
              <StubBanner title="Validate unavailable" message={validateError.message} />
            ) : (
              <ErrorBlock title="Could not validate changes" message={validateError.message} />
            )
          ) : null}

          {publishError ? (
            publishError instanceof ApiError && publishError.status === 501 ? (
              <StubBanner title="Publish unavailable" message={publishError.message} />
            ) : (
              <ErrorBlock title="Could not publish campaign" message={publishError.message} />
            )
          ) : null}

          {publishSuccess ? (
            <Badge variant="secondary">Campaign published</Badge>
          ) : null}

          {publishCheck ? (
            <div className="admin-panel grid gap-3 p-3">
              <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-medium">Publish check</p>
                <ValidityBadge
                  valid={publishCheck.valid}
                  validLabel="Ready"
                  invalidLabel="Blocked"
                />
              </div>
              <FieldErrorsPanel title="Field errors" fieldErrors={publishCheck.field_errors} />
              <StringList title="Warnings" items={publishCheck.warning_slugs} />
            </div>
          ) : null}

          {validateResult ? (
            <div className="admin-panel grid gap-3 p-3">
              <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-medium">Patch validation</p>
                <ValidityBadge
                  valid={validateResult.valid}
                  validLabel="Valid"
                  invalidLabel="Invalid"
                />
              </div>
              <FieldErrorsPanel title="Field errors" fieldErrors={validateResult.field_errors} />
              <StringList title="Warnings" items={validateResult.warnings} />
            </div>
          ) : null}

          {publishBlocked ? (
            <div className="admin-panel grid gap-3 border border-destructive/50 p-3">
              <div className="flex flex-wrap items-center gap-2">
                <p className="text-sm font-medium text-destructive">Publish blocked</p>
                <Badge variant="destructive">422</Badge>
              </div>
              <FieldErrorsPanel title="Field errors" fieldErrors={publishBlocked.field_errors} />
              <StringList title="Warning slugs" items={publishBlocked.warning_slugs} />
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Sheet onOpenChange={setCloneOpen} open={cloneOpen}>
        <SheetContent className="overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Clone campaign</SheetTitle>
          </SheetHeader>
          <div className="grid gap-4 pt-4">
            <div className="grid gap-2">
              <Label htmlFor="campaign-clone-name-suffix">Clone name suffix</Label>
              <Input
                id="campaign-clone-name-suffix"
                value={cloneNameSuffix}
                disabled={clonePreviewing || cloning || fetching}
                placeholder=" (copy)"
                onChange={(event) => onCloneNameSuffixChange(event.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave empty for the default &quot;{campaign.name} (copy)&quot;. Enter a suffix such as
                &quot; - v2&quot; to append to the source name.
              </p>
            </div>

            <div className="grid gap-3">
              <p className="text-sm font-medium">Clone options</p>
              {CLONE_OPTION_FIELDS.map(({ field, label, description }) => {
                const inputId = `campaign-clone-option-${field}`;
                const defaultChecked = field === 'reset_spend' ? false : true;
                const checked = cloneOptions[field] ?? defaultChecked;

                return (
                  <div key={field} className="grid gap-1">
                    <div className="flex items-center gap-2">
                      <Checkbox
                        checked={checked}
                        disabled={clonePreviewing || cloning || fetching}
                        id={inputId}
                        onCheckedChange={(value) => onCloneOptionChange(field, value === true)}
                      />
                      <Label htmlFor={inputId}>{label}</Label>
                    </div>
                    <p className="text-xs text-muted-foreground">{description}</p>
                  </div>
                );
              })}
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button
                type="button"
                variant="secondary"
                disabled={clonePreviewing || cloning || fetching}
                onClick={onClonePreview}
              >
                {clonePreviewing ? 'Previewing...' : 'Preview clone'}
              </Button>
              <Button
                type="button"
                disabled={clonePreviewing || cloning || fetching}
                onClick={onCloneExecute}
              >
                {cloning ? 'Creating clone...' : 'Create clone'}
              </Button>
            </div>

            {cloneError ? (
              cloneError instanceof ApiError && cloneError.status === 501 ? (
                <StubBanner title="Clone unavailable" message={cloneError.message} />
              ) : (
                <ErrorBlock title="Could not create clone" message={cloneError.message} />
              )
            ) : null}

            {cloneSuccess && clonedCampaignId ? (
              <p className="text-sm text-muted-foreground">
                Clone created.{' '}
                <Link
                  className="text-primary hover:underline"
                  to={`/campaigns/${clonedCampaignId}/edit`}
                >
                  Open cloned campaign
                </Link>
              </p>
            ) : null}

            {clonePreviewError ? (
              clonePreviewError instanceof ApiError && clonePreviewError.status === 501 ? (
                <StubBanner title="Clone preview unavailable" message={clonePreviewError.message} />
              ) : (
                <ErrorBlock title="Could not preview clone" message={clonePreviewError.message} />
              )
            ) : null}

            {clonePreview ? (
              <JsonDashboardView payload={clonePreview as unknown as Record<string, unknown>} />
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

      {cloneSuccess && clonedCampaignId ? (
        <p className="text-sm text-muted-foreground">
          Clone created.{' '}
          <Link
            className="text-primary hover:underline"
            to={`/campaigns/${clonedCampaignId}/edit`}
          >
            Open cloned campaign
          </Link>
        </p>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Compare campaigns</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="campaign-diff-against-id">Against campaign ID</Label>
            <Input
              id="campaign-diff-against-id"
              value={diffAgainstId}
              disabled={comparingDiff || fetching}
              placeholder="Other campaign UUID"
              onChange={(event) => onDiffAgainstIdChange(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Compare this campaign ({campaign.id}) against another campaign in the same customer.
            </p>
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="secondary"
              disabled={comparingDiff || fetching}
              onClick={onCompareDiff}
            >
              {comparingDiff ? 'Comparing...' : 'Compare'}
            </Button>
          </div>

          {diffError ? (
            diffError instanceof ApiError && diffError.status === 501 ? (
              <StubBanner title="Campaign diff unavailable" message={diffError.message} />
            ) : (
              <ErrorBlock title="Could not compare campaigns" message={diffError.message} />
            )
          ) : null}

          {diffResult ? (
            <div className="grid gap-3">
              {diffResult.truncated ? (
                <Badge variant="outline">Diff truncated - showing first rows only</Badge>
              ) : null}
              {diffResult.rows.length === 0 ? (
                <p className="text-sm text-muted-foreground">No differences found.</p>
              ) : (
                <div className="ui-table-frame">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Field</TableHead>
                        <TableHead>This campaign</TableHead>
                        <TableHead>Against campaign</TableHead>
                        <TableHead>Severity</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {diffResult.rows.map((row) => (
                        <TableRow key={row.path}>
                          <TableCell className="font-medium">{row.label}</TableCell>
                          <TableCell className="font-mono text-xs">
                            {formatReadonly(row.left_display)}
                          </TableCell>
                          <TableCell className="font-mono text-xs">
                            {formatReadonly(row.right_display)}
                          </TableCell>
                          <TableCell>
                            <Badge variant={diffSeverityVariant(row.severity)}>
                              {row.severity}
                            </Badge>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Owner and export</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
            <div className="grid gap-2">
              <Label htmlFor="campaign-owner-user-id">New owner user ID</Label>
              <Input
                id="campaign-owner-user-id"
                value={draftOwnerUserId}
                disabled={transferringOwner || fetching}
                onChange={(event) => onDraftOwnerUserIdChange(event.target.value)}
              />
            </div>
            <Button
              type="button"
              disabled={transferringOwner || fetching}
              onClick={onTransferOwner}
            >
              {transferringOwner ? 'Transferring...' : 'Transfer owner'}
            </Button>
          </div>
          {ownerSuccess ? (
            <p className="text-sm text-muted-foreground" role="status">
              Owner transfer accepted.
            </p>
          ) : null}
          {ownerError ? (
            ownerError instanceof ApiError && ownerError.status === 501 ? (
              <StubBanner title="Owner transfer unavailable" message={ownerError.message} />
            ) : (
              <ErrorBlock title="Could not transfer owner" message={ownerError.message} />
            )
          ) : null}

          <div className="flex justify-start">
            <Button
              type="button"
              variant="secondary"
              disabled={exporting || fetching}
              onClick={onExportCampaign}
            >
              {exporting ? 'Exporting...' : 'Download export bundle'}
            </Button>
          </div>
          {exportError ? (
            exportError instanceof ApiError && exportError.status === 501 ? (
              <StubBanner title="Export unavailable" message={exportError.message} />
            ) : (
              <ErrorBlock title="Could not export campaign" message={exportError.message} />
            )
          ) : null}
        </CardContent>
      </Card>

            <CampaignEditorTools campaignId={campaign.id} />
          </div>
        }
        onClone={() => setCloneOpen(true)}
        onFieldChange={onFieldChange}
        onSave={onSave}
        onSaveAndClose={onSaveAndClose ?? onSave}
      />
    </>
  );
}
