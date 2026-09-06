import { Copy } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import { toast } from 'sonner';

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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  settingsBentoBodyCellClass,
  settingsBentoHeaderCellClass,
  settingsBentoMobileStackClass,
  settingsBentoTableClass,
  settingsBentoTypeClass,
  settingsColumnClass,
  settingsColumnPanelClass,
  settingsGridCellInnerClass,
  settingsRowLabelClass,
  settingsRowValueClass,
  settingsSectionTitleClass,
} from '@/domains/settings/settings_classes';
import { settingsTextValue } from '@/domains/settings/settings_empty';
import type { PlatformSettingsSnapshot } from '@/domains/settings/settings_snapshot';
import { settingsFieldLabel } from '@/lib/settings_labels';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

type SettingsRowProps = {
  className?: string;
  label: string;
  value: ReactNode;
};

function SettingsRow({ className, label, value }: SettingsRowProps) {
  return (
    <div className={cn(settingsGridCellInnerClass, 'px-3 py-2', className)}>
      <span className={settingsRowLabelClass}>{label}</span>
      <div className={settingsRowValueClass}>{value}</div>
    </div>
  );
}

function SettingsGridCell({ className, label, value }: SettingsRowProps) {
  return (
    <div className={cn(settingsGridCellInnerClass, className)}>
      <span className={settingsRowLabelClass}>{label}</span>
      <div className={settingsRowValueClass}>{value}</div>
    </div>
  );
}

function SettingsColumn({
  className,
  rows,
  title,
}: {
  className?: string;
  rows: SettingsRowProps[];
  title: string;
}) {
  return (
    <section className={cn(settingsColumnClass, className)}>
      <h3 className={cn(settingsSectionTitleClass, settingsBentoTypeClass)}>{title}</h3>
      <div className={settingsColumnPanelClass}>
        {rows.map((row) => (
          <SettingsRow key={row.label} label={row.label} value={row.value} />
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
  const variant = tone === 'positive' ? 'active' : tone === 'unknown' ? 'paused' : 'secondary';

  return (
    <Badge className="gap-1.5" variant={variant}>
      <span
        aria-hidden="true"
        className={cn(
          'h-1.5 w-1.5 rounded-full',
          tone === 'positive' && 'bg-admin-status-active',
          tone === 'neutral' && 'bg-muted-foreground/60',
          tone === 'unknown' && 'bg-admin-status-paused'
        )}
      />
      {label}
    </Badge>
  );
}

export function SettingsPatchSwitch({
  checked,
  disabled = false,
  field,
  label,
  onPatch,
  patching,
}: {
  checked: boolean | undefined;
  disabled?: boolean;
  field: 'telemetry_enabled' | 'edge_xdp' | 'edge_expose_click' | 'edge_expose_openrtb';
  label: string;
  onPatch: (patch: Record<string, unknown>) => void;
  patching: boolean;
}) {
  return (
    <Switch
      aria-label={label}
      checked={checked === true}
      disabled={disabled || patching}
      onCheckedChange={(next) => onPatch({ [field]: next })}
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
      <span
        className={cn(
          'inline-flex h-7 max-w-full items-center border border-dashed border-border px-2.5 text-xs text-muted-foreground',
          adminKit.controlRadius
        )}
      >
        {shortLabel} not set
      </span>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          className="h-7 max-w-full gap-1.5 px-2 text-xs"
          type="button"
          variant="outline"
          onClick={() => {
            void navigator.clipboard.writeText(trimmed).then(
              () => toast.success(`${label} copied`),
              () => toast.error('Could not copy to clipboard')
            );
          }}
        >
          <span className="whitespace-nowrap">{shortLabel}</span>
          <Copy className="h-3.5 w-3.5 shrink-0" aria-hidden />
        </Button>
      </TooltipTrigger>
      <TooltipContent className={cn('max-w-sm break-all leading-snug', settingsBentoTypeClass)}>
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
        <Button type="button" variant="outline">
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
            <div
              className={cn(
                'grid gap-2 border border-border bg-muted/30 p-3 text-xs text-muted-foreground',
                adminKit.panelRadius
              )}
            >
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
            <Label htmlFor="stripe-secret-key-input">
              {settingsFieldLabel('stripe_secret_key')}
            </Label>
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
  patching: boolean
): Array<{ title: string; rows: SettingsRowProps[] }> {
  const currency = snapshot.config.defaultCurrency.trim();
  const timezone = snapshot.config.timezone.trim();
  const localeLine =
    currency || timezone
      ? [currency || '-', timezone || '-'].join(' / ')
      : settingsTextValue('', 'default_currency');
  const networkInterface = snapshot.config.networkInterface.trim();

  return [
    {
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
            <SettingsPatchSwitch
              checked={snapshot.config.telemetryEnabled}
              field="telemetry_enabled"
              label={settingsFieldLabel('telemetry_enabled')}
              patching={patching}
              onPatch={onPatchPlatform}
            />
          ),
        },
        {
          label: settingsFieldLabel('network_interface'),
          value: networkInterface ? (
            <span className="break-all">{networkInterface}</span>
          ) : (
            settingsTextValue(networkInterface, 'network_interface')
          ),
        },
      ],
    },
    {
      title: 'Traffic and routing',
      rows: [
        {
          label: settingsFieldLabel('tracking_domain'),
          value: (
            <span className="break-all">
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
      title: 'Edge and integration',
      rows: [
        {
          label: settingsFieldLabel('edge_xdp'),
          value: (
            <SettingsPatchSwitch
              checked={snapshot.config.edgeXdp}
              field="edge_xdp"
              label={settingsFieldLabel('edge_xdp')}
              patching={patching}
              onPatch={onPatchPlatform}
            />
          ),
        },
        {
          label: settingsFieldLabel('edge_expose_click'),
          value: (
            <SettingsPatchSwitch
              checked={snapshot.config.edgeExposeClick}
              field="edge_expose_click"
              label={settingsFieldLabel('edge_expose_click')}
              patching={patching}
              onPatch={onPatchPlatform}
            />
          ),
        },
        {
          label: settingsFieldLabel('edge_expose_openrtb'),
          value: (
            <SettingsPatchSwitch
              checked={snapshot.config.edgeExposeOpenRTB}
              field="edge_expose_openrtb"
              label={settingsFieldLabel('edge_expose_openrtb')}
              patching={patching}
              onPatch={onPatchPlatform}
            />
          ),
        },
        {
          label: 'Stripe secrets',
          value: (
            <span className="inline-flex max-w-full flex-wrap items-center gap-2">
              <span className="tracking-widest text-muted-foreground">--------</span>
              <StripeSecretsDialog
                onSave={onPatchPlatform}
                patching={patching}
                snapshot={snapshot}
              />
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
  const rowCount = columns.reduce((max, column) => Math.max(max, column.rows.length), 0);

  return (
    <>
      <Table bare className={settingsBentoTableClass}>
        <colgroup>
          <col className="w-1/3" />
          <col className="w-1/3" />
          <col className="w-1/3" />
        </colgroup>
        <TableHeader>
          <TableRow>
            {columns.map((column) => (
              <TableHead key={column.title} className={settingsBentoHeaderCellClass} scope="col">
                {column.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: rowCount }, (_, rowIndex) => (
            <TableRow key={rowIndex}>
              {columns.map((column) => {
                const row = column.rows[rowIndex];
                return (
                  <TableCell key={column.title} className={settingsBentoBodyCellClass}>
                    {row ? <SettingsGridCell {...row} /> : null}
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <div className={settingsBentoMobileStackClass}>
        {columns.map((column) => (
          <SettingsColumn key={column.title} rows={column.rows} title={column.title} />
        ))}
      </div>
    </>
  );
}

// Legacy export for tests or external imports.
export function BentoRow(props: SettingsRowProps) {
  return <SettingsRow {...props} />;
}
