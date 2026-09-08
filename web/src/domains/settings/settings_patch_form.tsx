import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField } from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { SettingsFormActions, SettingsFormStack } from '@/domains/settings/settings_form_stack';
import { settingsHintClass } from '@/domains/settings/settings_classes';
import type { PlatformSettingsPatch } from '@/api/types';
import type { PlatformSettingsSnapshot } from '@/domains/settings/settings_snapshot';
import { settingsFieldLabel } from '@/lib/settings_labels';

const INGRESS_SCHEMA_OPTIONS = [
  { value: 'ad_event_processor_native', label: 'ad_event_processor_native' },
  { value: 'openrtb_3', label: 'openrtb_3' },
];

export type SettingsPatchFormProps = {
  snapshot: PlatformSettingsSnapshot;
  patching: boolean;
  patchError: Error | undefined;
  patchSuccess: boolean;
  onPatchPlatform: (patch: PlatformSettingsPatch) => void;
};

export function SettingsPatchForm({
  snapshot,
  patching,
  patchError,
  patchSuccess,
  onPatchPlatform,
}: SettingsPatchFormProps) {
  const [trackingDomain, setTrackingDomain] = useState(snapshot.config.trackingDomain);
  const [defaultCurrency, setDefaultCurrency] = useState(snapshot.config.defaultCurrency);
  const [timezone, setTimezone] = useState(snapshot.config.timezone);
  const [ingressSchema, setIngressSchema] = useState(snapshot.config.ingressSchema);
  const [profile, setProfile] = useState(snapshot.config.profile);
  const [networkInterface, setNetworkInterface] = useState(snapshot.config.networkInterface);
  const [telemetryEnabled, setTelemetryEnabled] = useState(snapshot.config.telemetryEnabled ?? false);
  const [edgeXdp, setEdgeXdp] = useState(snapshot.config.edgeXdp ?? false);
  const [edgeExposeClick, setEdgeExposeClick] = useState(snapshot.config.edgeExposeClick ?? false);
  const [edgeExposeOpenRtb, setEdgeExposeOpenRtb] = useState(
    snapshot.config.edgeExposeOpenRTB ?? false
  );
  const [stripeEnabled, setStripeEnabled] = useState(snapshot.config.stripeEnabled ?? false);
  const [stripeSuccessUrl, setStripeSuccessUrl] = useState(snapshot.config.stripeCheckoutSuccessUrl);
  const [stripeCancelUrl, setStripeCancelUrl] = useState(snapshot.config.stripeCheckoutCancelUrl);

  useEffect(() => {
    setTrackingDomain(snapshot.config.trackingDomain);
    setDefaultCurrency(snapshot.config.defaultCurrency);
    setTimezone(snapshot.config.timezone);
    setIngressSchema(snapshot.config.ingressSchema);
    setProfile(snapshot.config.profile);
    setNetworkInterface(snapshot.config.networkInterface);
    setTelemetryEnabled(snapshot.config.telemetryEnabled ?? false);
    setEdgeXdp(snapshot.config.edgeXdp ?? false);
    setEdgeExposeClick(snapshot.config.edgeExposeClick ?? false);
    setEdgeExposeOpenRtb(snapshot.config.edgeExposeOpenRTB ?? false);
    setStripeEnabled(snapshot.config.stripeEnabled ?? false);
    setStripeSuccessUrl(snapshot.config.stripeCheckoutSuccessUrl);
    setStripeCancelUrl(snapshot.config.stripeCheckoutCancelUrl);
  }, [snapshot]);

  const onSubmit = () => {
    const patch: PlatformSettingsPatch = {};
    if (trackingDomain.trim() !== snapshot.config.trackingDomain) {
      patch.tracking_domain = trackingDomain.trim();
    }
    if (defaultCurrency.trim() !== snapshot.config.defaultCurrency) {
      patch.default_currency = defaultCurrency.trim();
    }
    if (timezone.trim() !== snapshot.config.timezone) {
      patch.timezone = timezone.trim();
    }
    if (ingressSchema !== snapshot.config.ingressSchema) {
      patch.ingress_schema = ingressSchema as PlatformSettingsPatch['ingress_schema'];
    }
    if (profile.trim() !== snapshot.config.profile) {
      patch.profile = profile.trim();
    }
    if (networkInterface.trim() !== snapshot.config.networkInterface) {
      patch.network_interface = networkInterface.trim();
    }
    if (telemetryEnabled !== (snapshot.config.telemetryEnabled ?? false)) {
      patch.telemetry_enabled = telemetryEnabled;
    }
    if (edgeXdp !== (snapshot.config.edgeXdp ?? false)) {
      patch.edge_xdp = edgeXdp;
    }
    if (edgeExposeClick !== (snapshot.config.edgeExposeClick ?? false)) {
      patch.edge_expose_click = edgeExposeClick;
    }
    if (edgeExposeOpenRtb !== (snapshot.config.edgeExposeOpenRTB ?? false)) {
      patch.edge_expose_openrtb = edgeExposeOpenRtb;
    }

    const stripePatch: NonNullable<PlatformSettingsPatch['stripe']> = {};
    if (stripeEnabled !== (snapshot.config.stripeEnabled ?? false)) {
      stripePatch.enabled = stripeEnabled;
    }
    if (stripeSuccessUrl.trim() !== snapshot.config.stripeCheckoutSuccessUrl) {
      stripePatch.checkout_success_url = stripeSuccessUrl.trim();
    }
    if (stripeCancelUrl.trim() !== snapshot.config.stripeCheckoutCancelUrl) {
      stripePatch.checkout_cancel_url = stripeCancelUrl.trim();
    }
    if (Object.keys(stripePatch).length > 0) {
      patch.stripe = stripePatch;
    }

    if (Object.keys(patch).length === 0) {
      return;
    }
    onPatchPlatform(patch);
  };

  const hasChanges =
    trackingDomain.trim() !== snapshot.config.trackingDomain ||
    defaultCurrency.trim() !== snapshot.config.defaultCurrency ||
    timezone.trim() !== snapshot.config.timezone ||
    ingressSchema !== snapshot.config.ingressSchema ||
    profile.trim() !== snapshot.config.profile ||
    networkInterface.trim() !== snapshot.config.networkInterface ||
    telemetryEnabled !== (snapshot.config.telemetryEnabled ?? false) ||
    edgeXdp !== (snapshot.config.edgeXdp ?? false) ||
    edgeExposeClick !== (snapshot.config.edgeExposeClick ?? false) ||
    edgeExposeOpenRtb !== (snapshot.config.edgeExposeOpenRTB ?? false) ||
    stripeEnabled !== (snapshot.config.stripeEnabled ?? false) ||
    stripeSuccessUrl.trim() !== snapshot.config.stripeCheckoutSuccessUrl ||
    stripeCancelUrl.trim() !== snapshot.config.stripeCheckoutCancelUrl;

  return (
    <SettingsFormStack
      onSubmit={(event) => {
        event.preventDefault();
        if (!patching && hasChanges) {
          onSubmit();
        }
      }}
    >
      <p className={settingsHintClass}>
        Edit platform configuration fields. Only changed values are sent on apply.
      </p>
      <FilterField htmlFor="settings-tracking-domain" label={settingsFieldLabel('tracking_domain')}>
        <Input
          id="settings-tracking-domain"
          value={trackingDomain}
          onChange={(event) => setTrackingDomain(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-default-currency" label={settingsFieldLabel('default_currency')}>
        <Input
          id="settings-default-currency"
          value={defaultCurrency}
          onChange={(event) => setDefaultCurrency(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-timezone" label={settingsFieldLabel('timezone')}>
        <Input
          id="settings-timezone"
          value={timezone}
          onChange={(event) => setTimezone(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-ingress-schema" label={settingsFieldLabel('ingress_schema')}>
        <Select value={ingressSchema || INGRESS_SCHEMA_OPTIONS[0].value} onValueChange={setIngressSchema}>
          <SelectTrigger id="settings-ingress-schema" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {INGRESS_SCHEMA_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FilterField>
      <FilterField htmlFor="settings-profile" label={settingsFieldLabel('profile')}>
        <Input id="settings-profile" value={profile} onChange={(event) => setProfile(event.target.value)} />
      </FilterField>
      <FilterField htmlFor="settings-network-interface" label={settingsFieldLabel('network_interface')}>
        <Input
          id="settings-network-interface"
          value={networkInterface}
          onChange={(event) => setNetworkInterface(event.target.value)}
        />
      </FilterField>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-telemetry-enabled">{settingsFieldLabel('telemetry_enabled')}</Label>
          <Switch
            checked={telemetryEnabled}
            id="settings-telemetry-enabled"
            onCheckedChange={setTelemetryEnabled}
          />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-edge-xdp">{settingsFieldLabel('edge_xdp')}</Label>
          <Switch checked={edgeXdp} id="settings-edge-xdp" onCheckedChange={setEdgeXdp} />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-edge-expose-click">{settingsFieldLabel('edge_expose_click')}</Label>
          <Switch
            checked={edgeExposeClick}
            id="settings-edge-expose-click"
            onCheckedChange={setEdgeExposeClick}
          />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-edge-expose-openrtb">{settingsFieldLabel('edge_expose_openrtb')}</Label>
          <Switch
            checked={edgeExposeOpenRtb}
            id="settings-edge-expose-openrtb"
            onCheckedChange={setEdgeExposeOpenRtb}
          />
        </div>
      </div>
      <div className="grid gap-3 rounded-md border p-3">
        <div className="flex items-center justify-between gap-2">
          <Label htmlFor="settings-stripe-enabled">Stripe enabled</Label>
          <Switch
            checked={stripeEnabled}
            id="settings-stripe-enabled"
            onCheckedChange={setStripeEnabled}
          />
        </div>
        <FilterField htmlFor="settings-stripe-success-url" label="Stripe checkout success URL">
          <Input
            id="settings-stripe-success-url"
            value={stripeSuccessUrl}
            onChange={(event) => setStripeSuccessUrl(event.target.value)}
          />
        </FilterField>
        <FilterField htmlFor="settings-stripe-cancel-url" label="Stripe checkout cancel URL">
          <Input
            id="settings-stripe-cancel-url"
            value={stripeCancelUrl}
            onChange={(event) => setStripeCancelUrl(event.target.value)}
          />
        </FilterField>
      </div>
      <SettingsFormActions>
        <PrimaryActionButton disabled={patching || !hasChanges} loading={patching} type="submit">
          {patching ? 'Applying...' : 'Apply changes'}
        </PrimaryActionButton>
        {patchSuccess ? (
          <p className={settingsHintClass} role="status">
            Configuration updated.
          </p>
        ) : null}
      </SettingsFormActions>
      {patchError ? <ErrorBlock title="Could not apply changes" message={patchError.message} /> : null}
    </SettingsFormStack>
  );
}
