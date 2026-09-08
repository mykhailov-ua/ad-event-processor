import { useState } from 'react';

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
import { settingsFieldLabel } from '@/lib/settings_labels';

const INGRESS_SCHEMA_OPTIONS = [
  { value: 'ad_event_processor_native', label: 'ad_event_processor_native' },
  { value: 'openrtb_3', label: 'openrtb_3' },
];

export type SettingsBootstrapDraft = {
  admin_email: string;
  admin_password: string;
  tracking_domain: string;
  default_currency: string;
  timezone: string;
  ingress_schema: string;
  telemetry_enabled: boolean;
  edge_xdp: boolean;
  edge_expose_click: boolean;
  edge_expose_openrtb: boolean;
  network_interface: string;
  license_key: string;
  license_server: string;
  deployment_id: string;
  eula_version: string;
};

export type SettingsBootstrapFormProps = {
  bootstrapping: boolean;
  bootstrapError: Error | undefined;
  bootstrapSuccess: boolean;
  draftInstallToken: string;
  onDraftInstallTokenChange: (value: string) => void;
  onRunBootstrap: (draft: SettingsBootstrapDraft) => void;
};

export function SettingsBootstrapForm({
  bootstrapping,
  bootstrapError,
  bootstrapSuccess,
  draftInstallToken,
  onDraftInstallTokenChange,
  onRunBootstrap,
}: SettingsBootstrapFormProps) {
  const [adminEmail, setAdminEmail] = useState('');
  const [adminPassword, setAdminPassword] = useState('');
  const [trackingDomain, setTrackingDomain] = useState('');
  const [defaultCurrency, setDefaultCurrency] = useState('USD');
  const [timezone, setTimezone] = useState('UTC');
  const [ingressSchema, setIngressSchema] = useState(INGRESS_SCHEMA_OPTIONS[0].value);
  const [telemetryEnabled, setTelemetryEnabled] = useState(true);
  const [edgeXdp, setEdgeXdp] = useState(false);
  const [edgeExposeClick, setEdgeExposeClick] = useState(true);
  const [edgeExposeOpenRtb, setEdgeExposeOpenRtb] = useState(false);
  const [networkInterface, setNetworkInterface] = useState('');
  const [licenseKey, setLicenseKey] = useState('');
  const [licenseServer, setLicenseServer] = useState('');
  const [deploymentId, setDeploymentId] = useState('');
  const [eulaVersion, setEulaVersion] = useState('');

  const canSubmit =
    draftInstallToken.trim() !== '' &&
    adminEmail.trim() !== '' &&
    adminPassword.trim() !== '' &&
    trackingDomain.trim() !== '';

  return (
    <SettingsFormStack
      onSubmit={(event) => {
        event.preventDefault();
        if (!bootstrapping && canSubmit) {
          onRunBootstrap({
            admin_email: adminEmail.trim(),
            admin_password: adminPassword,
            tracking_domain: trackingDomain.trim(),
            default_currency: defaultCurrency.trim(),
            timezone: timezone.trim(),
            ingress_schema: ingressSchema,
            telemetry_enabled: telemetryEnabled,
            edge_xdp: edgeXdp,
            edge_expose_click: edgeExposeClick,
            edge_expose_openrtb: edgeExposeOpenRtb,
            network_interface: networkInterface.trim(),
            license_key: licenseKey.trim(),
            license_server: licenseServer.trim(),
            deployment_id: deploymentId.trim(),
            eula_version: eulaVersion.trim(),
          });
        }
      }}
    >
      <p className={settingsHintClass}>
        Create the platform configuration on first run using the setup token from your deployment
        bundle.
      </p>
      <FilterField htmlFor="settings-install-token" label="Setup token">
        <Input
          id="settings-install-token"
          type="password"
          autoComplete="off"
          value={draftInstallToken}
          onChange={(event) => onDraftInstallTokenChange(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-admin-email" label="Admin email">
        <Input
          id="settings-admin-email"
          type="email"
          autoComplete="off"
          value={adminEmail}
          onChange={(event) => setAdminEmail(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-admin-password" label="Admin password">
        <Input
          id="settings-admin-password"
          type="password"
          autoComplete="new-password"
          value={adminPassword}
          onChange={(event) => setAdminPassword(event.target.value)}
        />
      </FilterField>
      <FilterField
        htmlFor="settings-bootstrap-tracking-domain"
        label={settingsFieldLabel('tracking_domain')}
      >
        <Input
          id="settings-bootstrap-tracking-domain"
          value={trackingDomain}
          onChange={(event) => setTrackingDomain(event.target.value)}
        />
      </FilterField>
      <FilterField
        htmlFor="settings-bootstrap-currency"
        label={settingsFieldLabel('default_currency')}
      >
        <Input
          id="settings-bootstrap-currency"
          value={defaultCurrency}
          onChange={(event) => setDefaultCurrency(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-bootstrap-timezone" label={settingsFieldLabel('timezone')}>
        <Input
          id="settings-bootstrap-timezone"
          value={timezone}
          onChange={(event) => setTimezone(event.target.value)}
        />
      </FilterField>
      <FilterField
        htmlFor="settings-bootstrap-ingress-schema"
        label={settingsFieldLabel('ingress_schema')}
      >
        <Select value={ingressSchema} onValueChange={setIngressSchema}>
          <SelectTrigger id="settings-bootstrap-ingress-schema" className="w-full">
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
      <FilterField
        htmlFor="settings-bootstrap-network-interface"
        label={settingsFieldLabel('network_interface')}
      >
        <Input
          id="settings-bootstrap-network-interface"
          value={networkInterface}
          onChange={(event) => setNetworkInterface(event.target.value)}
        />
      </FilterField>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-bootstrap-telemetry">
            {settingsFieldLabel('telemetry_enabled')}
          </Label>
          <Switch
            checked={telemetryEnabled}
            id="settings-bootstrap-telemetry"
            onCheckedChange={setTelemetryEnabled}
          />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-bootstrap-edge-xdp">{settingsFieldLabel('edge_xdp')}</Label>
          <Switch checked={edgeXdp} id="settings-bootstrap-edge-xdp" onCheckedChange={setEdgeXdp} />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-bootstrap-edge-click">
            {settingsFieldLabel('edge_expose_click')}
          </Label>
          <Switch
            checked={edgeExposeClick}
            id="settings-bootstrap-edge-click"
            onCheckedChange={setEdgeExposeClick}
          />
        </div>
        <div className="flex items-center justify-between gap-2 rounded-md border p-3">
          <Label htmlFor="settings-bootstrap-edge-openrtb">
            {settingsFieldLabel('edge_expose_openrtb')}
          </Label>
          <Switch
            checked={edgeExposeOpenRtb}
            id="settings-bootstrap-edge-openrtb"
            onCheckedChange={setEdgeExposeOpenRtb}
          />
        </div>
      </div>
      <FilterField htmlFor="settings-license-key" label="License key (optional)">
        <Input
          id="settings-license-key"
          value={licenseKey}
          onChange={(event) => setLicenseKey(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-license-server" label="License server (optional)">
        <Input
          id="settings-license-server"
          value={licenseServer}
          onChange={(event) => setLicenseServer(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-deployment-id" label="Deployment ID (optional)">
        <Input
          id="settings-deployment-id"
          value={deploymentId}
          onChange={(event) => setDeploymentId(event.target.value)}
        />
      </FilterField>
      <FilterField htmlFor="settings-eula-version" label="EULA version (optional)">
        <Input
          id="settings-eula-version"
          value={eulaVersion}
          onChange={(event) => setEulaVersion(event.target.value)}
        />
      </FilterField>
      <SettingsFormActions>
        <PrimaryActionButton
          disabled={bootstrapping || !canSubmit}
          loading={bootstrapping}
          type="submit"
        >
          {bootstrapping ? 'Setting up...' : 'Complete setup'}
        </PrimaryActionButton>
        {bootstrapSuccess ? (
          <p className={settingsHintClass} role="status">
            Initial setup completed.
          </p>
        ) : null}
      </SettingsFormActions>
      {bootstrapError ? <ErrorBlock title="Setup failed" message={bootstrapError.message} /> : null}
    </SettingsFormStack>
  );
}
