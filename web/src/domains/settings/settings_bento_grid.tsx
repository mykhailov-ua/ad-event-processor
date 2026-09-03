import { Activity, Globe, Server, Shield } from 'lucide-react';
import { useState, type ReactNode } from 'react';

import { CopyButton } from '@/shell/copy_button';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { settingsFieldLabel } from '@/lib/settings_labels';
import { settingsTextValue } from '@/domains/settings/settings_empty';
import type { PlatformSettingsSnapshot } from '@/domains/settings/settings_snapshot';
import { cn } from '@/lib/utils';

const BENTO_ROW_CLASS = 'flex h-[3.25rem] items-center justify-between gap-3 px-4';

export function BentoColumn({
  children,
  icon,
  title,
}: {
  children: ReactNode;
  icon: ReactNode;
  title: string;
}) {
  return (
    <div className="flex min-w-0 flex-col">
      <div className="flex items-center gap-2 border-b border-border/40 px-4 py-2.5 text-[11px] font-medium tracking-wide text-muted-foreground">
        {icon}
        <span>{title}</span>
      </div>
      <div className="grid flex-1 divide-y divide-border/40">{children}</div>
    </div>
  );
}

export function BentoRow({
  label,
  value,
}: {
  label: string;
  value: ReactNode;
}) {
  return (
    <div className={BENTO_ROW_CLASS}>
      <span className="shrink-0 text-sm text-muted-foreground">{label}</span>
      <div className="flex min-w-0 shrink-0 items-center justify-end gap-2 text-sm">{value}</div>
    </div>
  );
}

export function SettingsStatusBadge({
  label,
  tone,
}: {
  label: string;
  tone: 'positive' | 'neutral' | 'unknown';
}) {
  return (
    <Badge className="gap-1.5" variant={tone === 'positive' ? 'default' : 'secondary'}>
      <span
        aria-hidden="true"
        className={cn(
          'h-1.5 w-1.5 rounded-full',
          tone === 'positive' && 'bg-emerald-400',
          tone === 'neutral' && 'bg-muted-foreground/60',
          tone === 'unknown' && 'bg-amber-500',
        )}
      />
      {label}
    </Badge>
  );
}

export function SettingsReadOnlySwitch({
  checked,
  label,
}: {
  checked: boolean | undefined;
  label: string;
}) {
  return (
    <Switch
      aria-label={label}
      aria-readonly="true"
      checked={checked === true}
      className="pointer-events-none"
      tabIndex={-1}
    />
  );
}

function SettingsUrlCopyChip({
  label,
  shortLabel,
  value,
}: {
  label: string;
  shortLabel: string;
  value: string;
}) {
  const trimmed = value.trim();
  if (!trimmed) {
    return (
      <span className="inline-flex h-7 items-center rounded-sm border border-dashed border-border/60 px-2.5 text-xs text-muted-foreground">
        {shortLabel}
      </span>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex h-7 items-center gap-0.5 rounded-sm border border-border/60 bg-muted/25 pl-2.5 pr-0.5 text-xs text-foreground">
          <span>{shortLabel}</span>
          <CopyButton className="size-7" label={label} value={trimmed} />
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-sm break-all font-mono text-[11px] leading-snug">
        {trimmed}
      </TooltipContent>
    </Tooltip>
  );
}

export function SettingsUrlTemplateChips({
  clickUrl,
  openRtbUrl,
}: {
  clickUrl: string;
  openRtbUrl: string;
}) {
  return (
    <span className="inline-flex max-w-full flex-wrap items-center justify-end gap-1.5">
      <SettingsUrlCopyChip
        label={settingsFieldLabel('click_url_template')}
        shortLabel="Click"
        value={clickUrl}
      />
      <SettingsUrlCopyChip
        label={settingsFieldLabel('openrtb_endpoint_template')}
        shortLabel="OpenRTB"
        value={openRtbUrl}
      />
    </span>
  );
}

function StripeSecretsDialog({
  onSave,
  patching,
  snapshot,
}: {
  onSave: (patch: Record<string, unknown>) => void;
  patching: boolean;
  snapshot: PlatformSettingsSnapshot;
}) {
  const [open, setOpen] = useState(false);
  const [secretKey, setSecretKey] = useState('');
  const [webhookSecret, setWebhookSecret] = useState('');

  const hasStored =
    snapshot.secrets.stripeSecretKey.trim().length > 0 ||
    snapshot.secrets.stripeWebhookSecret.trim().length > 0;

  const onSubmit = () => {
    const stripe: Record<string, string> = {};
    if (secretKey.trim()) {
      stripe.secret_key = secretKey.trim();
    }
    if (webhookSecret.trim()) {
      stripe.webhook_secret = webhookSecret.trim();
    }
    if (Object.keys(stripe).length === 0) {
      return;
    }
    onSave({ stripe });
    setSecretKey('');
    setWebhookSecret('');
    setOpen(false);
  };

  return (
    <Dialog
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) {
          setSecretKey('');
          setWebhookSecret('');
        }
      }}
      open={open}
    >
      <DialogTrigger asChild>
        <Button className="px-3 text-xs" type="button" variant="outline">
          Configure
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Stripe secrets</DialogTitle>
          <DialogDescription>
            {hasStored
              ? 'Enter new values to rotate keys. Only filled fields are sent to the control plane.'
              : 'Add Stripe API credentials. Values are applied through the platform settings patch API.'}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          {hasStored ? (
            <div className="grid gap-2 rounded-xl bg-muted/30 p-3 text-xs text-muted-foreground">
              <p>
                Current secret key:{' '}
                <span className="font-mono text-foreground">
                  {snapshot.secrets.stripeSecretKey || 'not set'}
                </span>
              </p>
              <p>
                Current webhook secret:{' '}
                <span className="font-mono text-foreground">
                  {snapshot.secrets.stripeWebhookSecret || 'not set'}
                </span>
              </p>
            </div>
          ) : null}
          <div className="grid gap-2">
            <Label htmlFor="stripe-secret-key-input">{settingsFieldLabel('stripe_secret_key')}</Label>
            <Input
              autoComplete="off"
              id="stripe-secret-key-input"
              type="password"
              value={secretKey}
              onChange={(event) => setSecretKey(event.target.value)}
              placeholder="sk_live_..."
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="stripe-webhook-secret-input">
              {settingsFieldLabel('stripe_webhook_secret')}
            </Label>
            <Input
              autoComplete="off"
              id="stripe-webhook-secret-input"
              type="password"
              value={webhookSecret}
              onChange={(event) => setWebhookSecret(event.target.value)}
              placeholder="whsec_..."
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            disabled={patching || (!secretKey.trim() && !webhookSecret.trim())}
            onClick={onSubmit}
            type="button"
          >
            {patching ? 'Saving...' : 'Save secrets'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function SettingsBentoGrid({
  onPatchPlatform,
  patching,
  snapshot,
}: {
  onPatchPlatform: (patch: Record<string, unknown>) => void;
  patching: boolean;
  snapshot: PlatformSettingsSnapshot;
}) {
  const currency = snapshot.config.defaultCurrency.trim();
  const timezone = snapshot.config.timezone.trim();
  const localeLine =
    currency || timezone
      ? [currency || '-', timezone || '-'].join('  /  ')
      : settingsTextValue('', 'default_currency');

  const networkInterface = snapshot.config.networkInterface.trim();

  return (
    <div className="grid w-full md:grid-cols-3 md:divide-x md:divide-border/40">
      <BentoColumn icon={<Server className="h-3.5 w-3.5" />} title="Host and system">
        <BentoRow
          label={settingsFieldLabel('bootstrap_complete')}
          value={
            <SettingsStatusBadge
              label={snapshot.bootstrapComplete ? 'Complete' : 'Pending'}
              tone={snapshot.bootstrapComplete ? 'positive' : 'unknown'}
            />
          }
        />
        <BentoRow
          label={settingsFieldLabel('profile')}
          value={settingsTextValue(snapshot.config.profile, 'profile')}
        />
        <BentoRow
          label={settingsFieldLabel('telemetry_enabled')}
          value={
            <SettingsReadOnlySwitch
              checked={snapshot.config.telemetryEnabled}
              label={settingsFieldLabel('telemetry_enabled')}
            />
          }
        />
        <BentoRow
          label={settingsFieldLabel('network_interface')}
          value={
            networkInterface ? (
              <span className="inline-flex items-center gap-2 font-mono text-xs">
                <Activity className="h-3.5 w-3.5 text-emerald-500" />
                {networkInterface}
              </span>
            ) : (
              settingsTextValue(networkInterface, 'network_interface')
            )
          }
        />
      </BentoColumn>

      <BentoColumn icon={<Globe className="h-3.5 w-3.5" />} title="Traffic and routing">
        <BentoRow
          label={settingsFieldLabel('tracking_domain')}
          value={
            <span className="font-mono text-xs">
              {settingsTextValue(snapshot.config.trackingDomain, 'tracking_domain')}
            </span>
          }
        />
        <BentoRow
          label={settingsFieldLabel('ingress_schema')}
          value={settingsTextValue(snapshot.config.ingressSchema, 'ingress_schema')}
        />
        <BentoRow label="Locale" value={<span className="tabular-nums">{localeLine}</span>} />
        <BentoRow
          label="URL templates"
          value={
            <SettingsUrlTemplateChips
              clickUrl={snapshot.clickUrlTemplate}
              openRtbUrl={snapshot.openRtbEndpointTemplate}
            />
          }
        />
      </BentoColumn>

      <BentoColumn icon={<Shield className="h-3.5 w-3.5" />} title="Edge and integration">
        <BentoRow
          label={settingsFieldLabel('edge_xdp')}
          value={
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeXdp}
              label={settingsFieldLabel('edge_xdp')}
            />
          }
        />
        <BentoRow
          label={settingsFieldLabel('edge_expose_click')}
          value={
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeExposeClick}
              label={settingsFieldLabel('edge_expose_click')}
            />
          }
        />
        <BentoRow
          label={settingsFieldLabel('edge_expose_openrtb')}
          value={
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeExposeOpenRTB}
              label={settingsFieldLabel('edge_expose_openrtb')}
            />
          }
        />
        <BentoRow
          label="Stripe secrets"
          value={
            <span className="inline-flex items-center gap-2">
              <span className="font-mono text-xs tracking-widest text-muted-foreground">--------</span>
              <StripeSecretsDialog onSave={onPatchPlatform} patching={patching} snapshot={snapshot} />
            </span>
          }
        />
      </BentoColumn>
    </div>
  );
}
