import { Activity, Globe, Server, Shield } from 'lucide-react';
import { Fragment, useState, type CSSProperties, type ReactNode } from 'react';

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

type BentoCell = {
  label: string;
  value: ReactNode;
};

type BentoColumnDef = {
  icon: ReactNode;
  title: string;
  rows: BentoCell[];
};

const BENTO_ROW_CLASS =
  'grid min-h-0 min-w-0 grid-cols-1 gap-1 border-b border-border/40 px-4 py-3 last:border-b-0 sm:grid-cols-[minmax(0,9.5rem)_minmax(0,1fr)] sm:items-start sm:gap-x-3 sm:gap-y-1';

const BENTO_HEADER_CLASS =
  'flex min-w-0 items-center gap-2 border-b border-border/40 bg-muted/20 px-4 py-2.5 text-admin-caption font-medium tracking-wide text-muted-foreground';

export function BentoRow({
  className,
  label,
  style,
  value,
}: {
  className?: string;
  label: string;
  style?: CSSProperties;
  value: ReactNode;
}) {
  return (
    <div className={cn(BENTO_ROW_CLASS, className)} style={style}>
      <span className="min-w-0 break-words text-xs leading-snug text-muted-foreground">{label}</span>
      <div className="flex min-w-0 flex-wrap items-center gap-2 text-sm text-foreground sm:justify-self-end">
        {value}
      </div>
    </div>
  );
}

function BentoColumnHeader({
  className,
  icon,
  style,
  title,
}: {
  className?: string;
  icon: ReactNode;
  style?: React.CSSProperties;
  title: string;
}) {
  return (
    <div className={cn(BENTO_HEADER_CLASS, className)} style={style}>
      {icon}
      <span className="min-w-0 break-words">{title}</span>
    </div>
  );
}

function BentoColumnStack({
  className,
  icon,
  rows,
  title,
}: {
  className?: string;
  icon: ReactNode;
  rows: BentoCell[];
  title: string;
}) {
  return (
    <section className={cn('min-w-0', className)}>
      <BentoColumnHeader icon={icon} title={title} />
      <div className="min-w-0 divide-y divide-border/40">
        {rows.map((row) => (
          <BentoRow key={row.label} label={row.label} value={row.value} />
        ))}
      </div>
    </section>
  );
}

export function SettingsStatusBadge({
  label,
  tone,
}: {
  label: string;
  tone: 'positive' | 'neutral' | 'unknown';
}) {
  const variant =
    tone === 'positive' ? 'active' : tone === 'unknown' ? 'paused' : 'secondary';

  return (
    <Badge className="gap-1.5" variant={variant}>
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
      className="pointer-events-none shrink-0"
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
      <span className="inline-flex h-7 max-w-full items-center rounded-sm border border-dashed border-border/60 px-2.5 text-xs text-muted-foreground">
        {shortLabel}
      </span>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex h-7 max-w-full items-center gap-0.5 rounded-sm border border-border/60 bg-muted/25 pl-2.5 pr-0.5 text-xs text-foreground">
          <span className="truncate">{shortLabel}</span>
          <CopyButton className="size-7 shrink-0" label={label} value={trimmed} />
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-sm break-all font-mono text-admin-caption leading-snug">
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
    <span className="inline-flex max-w-full flex-wrap items-center gap-1.5">
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
        <Button className="shrink-0 px-3 text-xs" type="button" variant="outline">
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

function buildColumns(
  snapshot: PlatformSettingsSnapshot,
  onPatchPlatform: (patch: Record<string, unknown>) => void,
  patching: boolean,
): BentoColumnDef[] {
  const currency = snapshot.config.defaultCurrency.trim();
  const timezone = snapshot.config.timezone.trim();
  const localeLine =
    currency || timezone
      ? [currency || '-', timezone || '-'].join('  /  ')
      : settingsTextValue('', 'default_currency');
  const networkInterface = snapshot.config.networkInterface.trim();

  return [
    {
      icon: <Server className="h-3.5 w-3.5 shrink-0" />,
      title: 'Host and system',
      rows: [
        {
          label: settingsFieldLabel('bootstrap_complete'),
          value: (
            <SettingsStatusBadge
              label={snapshot.bootstrapComplete ? 'Complete' : 'Pending'}
              tone={snapshot.bootstrapComplete ? 'positive' : 'unknown'}
            />
          ),
        },
        {
          label: settingsFieldLabel('profile'),
          value: settingsTextValue(snapshot.config.profile, 'profile'),
        },
        {
          label: settingsFieldLabel('telemetry_enabled'),
          value: (
            <SettingsReadOnlySwitch
              checked={snapshot.config.telemetryEnabled}
              label={settingsFieldLabel('telemetry_enabled')}
            />
          ),
        },
        {
          label: settingsFieldLabel('network_interface'),
          value: networkInterface ? (
            <span className="inline-flex max-w-full flex-wrap items-center gap-2 break-all font-mono text-xs">
              <Activity className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
              {networkInterface}
            </span>
          ) : (
            settingsTextValue(networkInterface, 'network_interface')
          ),
        },
      ],
    },
    {
      icon: <Globe className="h-3.5 w-3.5 shrink-0" />,
      title: 'Traffic and routing',
      rows: [
        {
          label: settingsFieldLabel('tracking_domain'),
          value: (
            <span className="break-all font-mono text-xs">
              {settingsTextValue(snapshot.config.trackingDomain, 'tracking_domain')}
            </span>
          ),
        },
        {
          label: settingsFieldLabel('ingress_schema'),
          value: settingsTextValue(snapshot.config.ingressSchema, 'ingress_schema'),
        },
        {
          label: 'Locale',
          value: <span className="break-words tabular-nums">{localeLine}</span>,
        },
        {
          label: 'URL templates',
          value: (
            <SettingsUrlTemplateChips
              clickUrl={snapshot.clickUrlTemplate}
              openRtbUrl={snapshot.openRtbEndpointTemplate}
            />
          ),
        },
      ],
    },
    {
      icon: <Shield className="h-3.5 w-3.5 shrink-0" />,
      title: 'Edge and integration',
      rows: [
        {
          label: settingsFieldLabel('edge_xdp'),
          value: (
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeXdp}
              label={settingsFieldLabel('edge_xdp')}
            />
          ),
        },
        {
          label: settingsFieldLabel('edge_expose_click'),
          value: (
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeExposeClick}
              label={settingsFieldLabel('edge_expose_click')}
            />
          ),
        },
        {
          label: settingsFieldLabel('edge_expose_openrtb'),
          value: (
            <SettingsReadOnlySwitch
              checked={snapshot.config.edgeExposeOpenRTB}
              label={settingsFieldLabel('edge_expose_openrtb')}
            />
          ),
        },
        {
          label: 'Stripe secrets',
          value: (
            <span className="inline-flex max-w-full flex-wrap items-center gap-2">
              <span className="font-mono text-xs tracking-widest text-muted-foreground">--------</span>
              <StripeSecretsDialog onSave={onPatchPlatform} patching={patching} snapshot={snapshot} />
            </span>
          ),
        },
      ],
    },
  ];
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
  const columns = buildColumns(snapshot, onPatchPlatform, patching);
  const rowCount = columns[0]?.rows.length ?? 0;

  return (
    <div className="min-w-0">
      <div className="grid min-w-0 divide-y divide-border/40 xl:hidden">
        {columns.map((column) => (
          <BentoColumnStack
            key={column.title}
            icon={column.icon}
            rows={column.rows}
            title={column.title}
          />
        ))}
      </div>

      <div
        className="hidden min-w-0 xl:grid xl:grid-cols-3"
        style={{ gridTemplateRows: `auto repeat(${rowCount}, minmax(0, auto))` }}
      >
        {columns.map((column, columnIndex) => (
          <Fragment key={column.title}>
            <BentoColumnHeader
              className={columnIndex > 0 ? 'border-l border-border/40' : undefined}
              icon={column.icon}
              title={column.title}
              style={{ gridColumn: columnIndex + 1, gridRow: 1 }}
            />
            {column.rows.map((row, rowIndex) => (
              <BentoRow
                key={row.label}
                className={cn(
                  columnIndex > 0 && 'border-l border-border/40',
                  rowIndex === column.rows.length - 1 && 'border-b-0',
                )}
                label={row.label}
                style={{ gridColumn: columnIndex + 1, gridRow: rowIndex + 2 }}
                value={row.value}
              />
            ))}
          </Fragment>
        ))}
      </div>
    </div>
  );
}
