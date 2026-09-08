import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor_types';
import { cn } from '@/lib/utils';

type CampaignEditorAdvancedRoutingSectionProps = {
  form: CampaignEditorFormState;
  saving: boolean;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K]
  ) => void;
};

export function CampaignEditorAdvancedRoutingSection({
  form,
  saving,
  onFieldChange,
}: CampaignEditorAdvancedRoutingSectionProps) {
  const { data: flows } = useResource((signal) => listFlows(signal), []);

  return (
    <>
      <section className="flex flex-col gap-4">
        <h2 className="text-sm font-semibold text-foreground">Routing & ingress</h2>
        <div className="grid gap-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
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
              {form.flow_id ? (
                <p className="text-xs text-muted-foreground font-mono">{form.flow_id}</p>
              ) : null}
            </div>
            <div className="grid gap-2">
              <Label htmlFor="campaign-brand-id">Brand ID</Label>
              <Input
                className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
                id="campaign-brand-id"
                value={form.brand_id}
                disabled={saving}
                onChange={(event) => onFieldChange('brand_id', event.target.value)}
              />
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
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
                <p className="text-xs text-amber-700 dark:text-amber-400">
                  Non-full tiers skip fraud filters and click budget debit. Tracker escalates to full
                  when fraud enforcement flags are enabled. redirect_only requires operator license
                  on the tracker (`CLICK_FILTER_REDIRECT_ONLY_LICENSED`).
                </p>
              ) : (
                <p className="text-xs text-muted-foreground">
                  Default production path: full FilterEngine and unified budget debit on /click.
                </p>
              )}
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-param">Ingress cost param</Label>
              <Input
                className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
                id="campaign-ingress-param"
                value={form.ingress_param}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_param', event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="campaign-ingress-scale">Ingress cost scale</Label>
              <Input
                className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
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
                className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
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
                className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
                id="campaign-ingress-policy"
                value={form.ingress_policy}
                disabled={saving}
                onChange={(event) => onFieldChange('ingress_policy', event.target.value)}
              />
            </div>
          </div>
        </div>
      </section>

      <section className="flex flex-col gap-4">
        <h2 className="text-sm font-semibold text-foreground">Integrations</h2>
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="campaign-traffic-template-id">Traffic template ID</Label>
            <Input
              className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
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
            <Textarea
              className={cn(
                CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MONO_CLASS,
                CAMPAIGN_CLICK_QUERY_PARAMS_TEXTAREA_MAX_HEIGHT_CLASS,
                'resize-y overflow-y-auto focus-visible:ring-0 focus-visible:ring-offset-0'
              )}
              disabled={saving}
              id="campaign-click-query-params"
              maxLength={CAMPAIGN_CLICK_QUERY_PARAMS_JSON_MAX_CHARS}
              placeholder="{}"
              rows={8}
              showCount
              value={form.click_query_params_json}
              onChange={(event) => onFieldChange('click_query_params_json', event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Query param macros for the click URL preset (sub1..sub30, ad_campaign_id, click ids).
              Values must be strings. {CAMPAIGN_CLICK_QUERY_PARAMS_FIELD_HINT}
            </p>
          </div>
        </div>
      </section>
    </>
  );
}
