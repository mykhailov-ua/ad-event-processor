import { Link } from 'react-router-dom';

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
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { CampaignEditorTools } from '@/domains/campaigns/editor/campaign_editor_tools';
import type { Campaign } from '@/api/types';
import type { CampaignEditorProps } from '@/domains/campaigns/editor/campaign_editor_types';
import {
  CLONE_OPTION_FIELDS,
  FieldErrorsPanel,
  StringList,
  ValidityBadge,
  diffSeverityVariant,
  editorApiErrorBlock,
  formatReadonly,
} from '@/domains/campaigns/editor/campaign_editor_shared';

export type CampaignEditorAdvancedPanelProps = Pick<
  CampaignEditorProps,
  | 'form'
  | 'saving'
  | 'fetching'
  | 'onFieldChange'
  | 'checking'
  | 'validating'
  | 'publishing'
  | 'forcePublish'
  | 'publishCheck'
  | 'validateResult'
  | 'publishBlocked'
  | 'publishSuccess'
  | 'publishCheckError'
  | 'validateError'
  | 'publishError'
  | 'onForcePublishChange'
  | 'onCheckPublish'
  | 'onValidateChanges'
  | 'onPublish'
  | 'macroPreviewForm'
  | 'onMacroPreviewFieldChange'
  | 'macroPreviewing'
  | 'macroPreviewResult'
  | 'macroPreviewError'
  | 'onMacroPreview'
  | 'cloneNameSuffix'
  | 'onCloneNameSuffixChange'
  | 'cloneOptions'
  | 'onCloneOptionChange'
  | 'clonePreviewing'
  | 'clonePreview'
  | 'clonePreviewError'
  | 'onClonePreview'
  | 'cloning'
  | 'cloneSuccess'
  | 'clonedCampaignId'
  | 'cloneError'
  | 'onCloneExecute'
  | 'diffAgainstId'
  | 'onDiffAgainstIdChange'
  | 'comparingDiff'
  | 'diffResult'
  | 'diffError'
  | 'onCompareDiff'
  | 'draftOwnerUserId'
  | 'onDraftOwnerUserIdChange'
  | 'transferringOwner'
  | 'ownerError'
  | 'ownerSuccess'
  | 'onTransferOwner'
  | 'exporting'
  | 'exportError'
  | 'onExportCampaign'
> & {
  campaign: Campaign;
  statusLabel: string;
  gateBusy: boolean;
  cloneOpen: boolean;
  onCloneOpenChange: (open: boolean) => void;
};

export function CampaignEditorAdvancedPanel({
  campaign,
  form,
  saving,
  fetching,
  onFieldChange,
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
  statusLabel,
  gateBusy,
  cloneOpen,
  onCloneOpenChange,
}: CampaignEditorAdvancedPanelProps) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-muted-foreground">
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

          {macroPreviewError
            ? editorApiErrorBlock(
                macroPreviewError,
                'Macro preview unavailable',
                'Could not preview macros',
              )
            : null}

          {macroPreviewResult ? (
            <div className="grid gap-3 rounded-md border border-border bg-card p-3">
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
            <Button type="button" disabled={gateBusy || fetching || saving} onClick={onPublish}>
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

          {publishCheckError
            ? editorApiErrorBlock(
                publishCheckError,
                'Publish check unavailable',
                'Could not check publish gate',
              )
            : null}
          {validateError
            ? editorApiErrorBlock(validateError, 'Validate unavailable', 'Could not validate changes')
            : null}
          {publishError
            ? editorApiErrorBlock(publishError, 'Publish unavailable', 'Could not publish campaign')
            : null}

          {publishSuccess ? <Badge variant="secondary">Campaign published</Badge> : null}

          {publishCheck ? (
            <div className="grid gap-3 rounded-md border border-border bg-card p-3">
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
            <div className="grid gap-3 rounded-md border border-border bg-card p-3">
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
            <div className="grid gap-3 rounded-md border border-destructive/50 bg-card p-3">
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

      <Sheet onOpenChange={onCloneOpenChange} open={cloneOpen}>
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

            {cloneError
              ? editorApiErrorBlock(cloneError, 'Clone unavailable', 'Could not create clone')
              : null}

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

            {clonePreviewError
              ? editorApiErrorBlock(
                  clonePreviewError,
                  'Clone preview unavailable',
                  'Could not preview clone',
                )
              : null}

            {clonePreview ? (
              <JsonDashboardView payload={clonePreview as unknown as Record<string, unknown>} />
            ) : null}
          </div>
        </SheetContent>
      </Sheet>

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

          {diffError
            ? editorApiErrorBlock(diffError, 'Campaign diff unavailable', 'Could not compare campaigns')
            : null}

          {diffResult ? (
            <div className="grid gap-3">
              {diffResult.truncated ? (
                <Badge variant="outline">Diff truncated - showing first rows only</Badge>
              ) : null}
              {diffResult.rows.length === 0 ? (
                <p className="text-sm text-muted-foreground">No differences found.</p>
              ) : (
                <DirectoryTable>
                  <TableHeader>
                    <TableRow>
                      <DirectoryTableHead>Field</DirectoryTableHead>
                      <DirectoryTableHead>This campaign</DirectoryTableHead>
                      <DirectoryTableHead>Against campaign</DirectoryTableHead>
                      <DirectoryTableHead>Severity</DirectoryTableHead>
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
                          <Badge variant={diffSeverityVariant(row.severity)}>{row.severity}</Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </DirectoryTable>
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
          {ownerError
            ? editorApiErrorBlock(
                ownerError,
                'Owner transfer unavailable',
                'Could not transfer owner',
              )
            : null}

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
          {exportError
            ? editorApiErrorBlock(exportError, 'Export unavailable', 'Could not export campaign')
            : null}
        </CardContent>
      </Card>

      <CampaignEditorTools campaignId={campaign.id} />
    </div>
  );
}
