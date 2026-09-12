import { Link } from 'react-router-dom';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { listFlows } from '@/api/flows_api';
import { useResource } from '@/api/use_resource';
import {
  CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT,
  CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS,
  CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MAX_HEIGHT_CLASS,
  CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MONO_CLASS,
  CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS,
} from '@/domains/campaigns/editor/campaign_click_query_limits';
import type { Campaign } from '@/api/types';
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor_types';
import { FraudLimitsDocLink } from '@/domains/campaigns/editor/fraud_limits_doc_link';
import { RedirectComplianceDocLink } from '@/domains/campaigns/editor/redirect_compliance_doc_link';
import { cn } from '@/lib/utils';

function sandboxPreviewPath(decoyLanderId: string, safePageUrl: string | undefined): string | null {
  const id = decoyLanderId.trim();
  if (id) {
    return `/lp/${id}/`;
  }
  const url = safePageUrl?.trim() ?? '';
  const match = url.match(/\/lp\/([0-9a-f-]{36})/i);
  return match ? `/lp/${match[1]}/` : null;
}

type CampaignEditorAdvancedRoutingSectionProps = {
  campaign: Campaign | undefined;
  form: CampaignEditorFormState;
  saving: boolean;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K]
  ) => void;
};

export function CampaignEditorAdvancedRoutingSection({
  campaign,
  form,
  saving,
  onFieldChange,
}: CampaignEditorAdvancedRoutingSectionProps) {
  const sandboxPreview = sandboxPreviewPath(form.decoy_lander_id, campaign?.safe_page_url);
  const { data: flows } = useResource((signal) => listFlows(signal), []);

  return (
    <>
      <section>
        <h2>Routing & ingress</h2>
        <div>
          <div>
            <div>
              <Label htmlFor="campaign-flow-id">Flow</Label>
              <Select
                disabled={saving}
                value={form.flow_id || undefined}
                onValueChange={(value) => onFieldChange('flow_id', value)}
              >
                <SelectTrigger id="campaign-flow-id">
                  <SelectValue placeholder="Select flow" />
                </SelectTrigger>
                <SelectContent>
                  {(flows ?? []).map((flow) => (
                    <SelectItem key={flow.id} value={flow.id}>
                      {flow.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {form.flow_id ? <p>{form.flow_id}</p> : null}
            </div>
            <div>
              <Label htmlFor="campaign-brand-id">Brand ID</Label>
              <Input
                id="campaign-brand-id"
                value={form.brand_id}
                disabled={saving}
                onChange={(event) => onFieldChange('brand_id', event.target.value)}
              />
            </div>
          </div>

          <div>
            <div>
              <Label htmlFor="campaign-redirect-compliance-mode">Redirect compliance</Label>
              <Select
                disabled={saving}
                value={form.redirect_compliance_mode || 'strict'}
                onValueChange={(value) => onFieldChange('redirect_compliance_mode', value)}
              >
                <SelectTrigger id="campaign-redirect-compliance-mode">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="strict">Strict (302 only)</SelectItem>
                  <SelectItem value="legacy_dmr">Legacy DMR</SelectItem>
                </SelectContent>
              </Select>
              <p>
                Strict is the default for new campaigns. Legacy DMR uses 200 meta refresh and may
                fingerprint redirect chains.
              </p>
              <RedirectComplianceDocLink />
            </div>
            <div>
              <div>
                <Checkbox
                  checked={form.dmr_enabled}
                  disabled={saving || form.redirect_compliance_mode !== 'legacy_dmr'}
                  id="campaign-dmr-enabled"
                  onCheckedChange={(checked) => onFieldChange('dmr_enabled', checked === true)}
                />
                <div>
                  <Label htmlFor="campaign-dmr-enabled">DMR on campaign clicks</Label>
                  <p>
                    Requires legacy DMR profile. Also available per click via dmr=1 on the click
                    URL.
                  </p>
                  {form.redirect_compliance_mode === 'legacy_dmr' ? (
                    <p>Warning: legacy DMR increases redirect-chain observability for scanners.</p>
                  ) : null}
                </div>
              </div>
            </div>
          </div>

          <div>
            <div>
              <Label htmlFor="campaign-click-filter-tier">Click filter tier</Label>
              <Select
                disabled={saving}
                value={form.click_filter_tier || 'full'}
                onValueChange={(value) => onFieldChange('click_filter_tier', value)}
              >
                <SelectTrigger id="campaign-click-filter-tier">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="full">Full (fraud + budget)</SelectItem>
                  <SelectItem value="light">Light (geo + license)</SelectItem>
                  <SelectItem value="redirect_only">Redirect only (fast TDS)</SelectItem>
                </SelectContent>
              </Select>
              {form.click_filter_tier !== 'full' ? (
                <p>
                  Non-full tiers skip fraud filters and click budget debit. Tracker escalates to
                  full when fraud enforcement flags are enabled. redirect_only requires operator
                  license on the tracker (`CLICK_FILTER_REDIRECT_ONLY_LICENSED`).
                </p>
              ) : (
                <p>
                  Default production path: full FilterEngine and unified budget debit on /click.
                </p>
              )}
            </div>
          </div>

          <div>
            <div>
              <Label htmlFor="campaign-ingress-param">Ingress cost param</Label>
              <Input
                id="campaign-ingress-param"
                value={form.ingress_param}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_param', event.target.value)}
              />
            </div>
            <div>
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

          <div>
            <div>
              <Label htmlFor="campaign-ingress-max-micro">Ingress max micro</Label>
              <Input
                id="campaign-ingress-max-micro"
                value={form.ingress_max_micro}
                disabled={saving}
                inputMode="numeric"
                onChange={(event) => onFieldChange('ingress_max_micro', event.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="campaign-ingress-policy">Ingress policy</Label>
              <Input
                id="campaign-ingress-policy"
                value={form.ingress_policy}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_policy', event.target.value)}
              />
            </div>
          </div>
        </div>
      </section>

      <section>
        <h2>Safe-page attestation</h2>
        <div>
          <div>
            <Checkbox
              checked={form.mobile_biometrics_click_enabled}
              disabled={saving}
              id="campaign-mobile-biometrics-click"
              onCheckedChange={(checked) =>
                onFieldChange('mobile_biometrics_click_enabled', checked === true)
              }
            />
            <div>
              <Label htmlFor="campaign-mobile-biometrics-click">
                Require mobile biometrics on click
              </Label>
              <p>
                Enforces gyro variance and touch pressure on safe-page verify before offer redirect.
                Requires tracker <span>MOBILE_BIOMETRICS_CLICK_ENABLED=1</span>, campaign safe-page
                + attestation, and probe JS sending devicemotion / touch force.
              </p>
              <FraudLimitsDocLink />
            </div>
          </div>
          <div>
            <div>
              <Label htmlFor="campaign-decoy-lander-id">Decoy lander ID</Label>
              <Input
                id="campaign-decoy-lander-id"
                value={form.decoy_lander_id}
                disabled={saving}
                placeholder="Hosted lander UUID"
                onChange={(event) => onFieldChange('decoy_lander_id', event.target.value)}
              />
              <p>
                Sandbox decoy uses the hosted lander shell at /lp/&#123;id&#125;/ for structural
                parity with production. When empty, the tracker derives from safe_page_url when it
                points at /lp/.
              </p>
              {form.decoy_lander_id.trim() ? (
                <p>
                  <Link
                    className="text-foreground underline"
                    to={`/landers/${encodeURIComponent(form.decoy_lander_id.trim())}/hosted-editor`}
                  >
                    Open hosted block editor
                  </Link>
                </p>
              ) : null}
            </div>
            {sandboxPreview ? (
              <div>
                <Label>Sandbox preview</Label>
                <p>{sandboxPreview}</p>
                <p>Open on the tracker origin to inspect decoy asset graph parity.</p>
              </div>
            ) : null}
          </div>
        </div>
      </section>

      <section>
        <h2>Integrations</h2>
        <div>
          <div>
            <Label htmlFor="campaign-traffic-template-id">Traffic template ID</Label>
            <Input
              id="campaign-traffic-template-id"
              value={form.traffic_template_id}
              disabled={saving}
              placeholder="meta-facebook"
              onChange={(event) => onFieldChange('traffic_template_id', event.target.value)}
            />
            <p>Integration click URL preset template (for example meta-facebook).</p>
          </div>

          <div>
            <Label htmlFor="campaign-click-query-params">Click query params (JSON)</Label>
            <Textarea
              disabled={saving}
              id="campaign-click-query-params"
              maxLength={CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS}
              placeholder="{}"
              rows={8}
              showCount
              value={form.click_query_params_json}
              onChange={(event) => onFieldChange('click_query_params_json', event.target.value)}
            />
            <p>
              Query param macros for the click URL preset (sub1..sub30, ad_campaign_id, click ids).
              Values must be strings. {CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT}
            </p>
          </div>
        </div>
      </section>
    </>
  );
}
