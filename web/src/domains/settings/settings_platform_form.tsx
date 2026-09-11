import type { ReactNode } from 'react';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password_input';
import type { MetaResponse, PlatformMaskedSecrets, PlatformSettingsView } from '@/api/types';
import type { PlatformSettingsDraft } from '@/domains/settings/settings_draft';
import { ingressSchemaLabel } from '@/domains/settings/settings_field_labels';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { FilterField, FilterPanel } from '@/shell/filter_panel';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { cn } from '@/lib/utils';

const FILTER_PANEL_FLAT = 'border-0 bg-transparent p-0 shadow-none';

export type SettingsPlatformFormProps = {
  meta: MetaResponse | undefined;
  platformSnapshot: PlatformSettingsView | undefined;
  maskedSecrets: PlatformMaskedSecrets | undefined;
  draft: PlatformSettingsDraft;
  draftDirty: boolean;
  canWrite: boolean;
  patching: boolean;
  patchError: Error | undefined;
  onDraftChange: (patch: Partial<PlatformSettingsDraft>) => void;
  onSave: () => void;
  onDiscard: () => void;
};

function PlatformSection({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className={cn('grid', adminSpacing.gap.lg)}>
      <h2 className={adminTypography.sectionTitle}>{title}</h2>
      {children}
    </section>
  );
}

export function SettingsPlatformForm({
  meta,
  platformSnapshot,
  maskedSecrets,
  draft,
  draftDirty,
  canWrite,
  patching,
  patchError,
  onDraftChange,
  onSave,
  onDiscard,
}: SettingsPlatformFormProps) {
  const ingressOptions = meta?.ingress_schemas ?? [];
  const profile = platformSnapshot?.config?.profile ?? '';
  const edgeXdpDisabled = profile === 'compose_dev';
  const paymentEnabled = meta?.payment_enabled === true;
  const saveDisabled = !canWrite || !draftDirty || patching;

  return (
    <div className={cn('grid', adminSpacing.gap.xl)}>
      {patchError ? <ErrorBlock error={patchError} title="Could not save platform settings" /> : null}

      <PlatformSection title="Tracker endpoints">
        <FilterPanel className={FILTER_PANEL_FLAT}>
          <FilterField htmlFor="settings-tracking-domain" label="Tracking domain">
            <Input
              disabled={!canWrite || patching}
              id="settings-tracking-domain"
              value={draft.trackingDomain}
              onChange={(event) => onDraftChange({ trackingDomain: event.target.value })}
            />
          </FilterField>
        </FilterPanel>
      </PlatformSection>

      <PlatformSection title="Edge and ingress">
        <FilterPanel className={FILTER_PANEL_FLAT}>
          <div className={cn('grid sm:grid-cols-2', adminSpacing.gap.xl)}>
            <FilterField htmlFor="settings-ingress-schema" label="Ingress schema">
              <Select
                disabled={!canWrite || patching || ingressOptions.length === 0}
                value={draft.ingressSchema || undefined}
                onValueChange={(value) => onDraftChange({ ingressSchema: value })}
              >
                <SelectTrigger id="settings-ingress-schema">
                  <SelectValue placeholder="Select schema" />
                </SelectTrigger>
                <SelectContent>
                  {ingressOptions.map((schema) => (
                    <SelectItem key={schema} value={schema}>
                      {ingressSchemaLabel(schema)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>
          </div>
          <div className={cn('grid', adminSpacing.gap.md)}>
            <label className={cn('flex items-center', adminSpacing.gap.md, adminTypography.body)}>
              <Checkbox
                checked={draft.edgeExposeClick}
                disabled={!canWrite || patching}
                onCheckedChange={(checked) =>
                  onDraftChange({ edgeExposeClick: checked === true })
                }
              />
              <span>Expose click endpoint on edge</span>
            </label>
            <label className={cn('flex items-center', adminSpacing.gap.md, adminTypography.body)}>
              <Checkbox
                checked={draft.edgeExposeOpenrtb}
                disabled={!canWrite || patching}
                onCheckedChange={(checked) =>
                  onDraftChange({ edgeExposeOpenrtb: checked === true })
                }
              />
              <span>Expose OpenRTB endpoint on edge</span>
            </label>
            <label className={cn('flex items-center', adminSpacing.gap.md, adminTypography.body)}>
              <Checkbox
                checked={draft.edgeXdp}
                disabled={!canWrite || patching || edgeXdpDisabled}
                onCheckedChange={(checked) => onDraftChange({ edgeXdp: checked === true })}
              />
              <span>
                Enable edge XDP
                {edgeXdpDisabled ? ' (not supported in compose_dev profile)' : ''}
              </span>
            </label>
          </div>
        </FilterPanel>
      </PlatformSection>

      <PlatformSection title="Regional defaults">
        <FilterPanel className={FILTER_PANEL_FLAT}>
          <div className={cn('grid sm:grid-cols-2', adminSpacing.gap.xl)}>
            <FilterField htmlFor="settings-default-currency" label="Default currency">
              <Input
                disabled={!canWrite || patching}
                id="settings-default-currency"
                maxLength={3}
                value={draft.defaultCurrency}
                onChange={(event) => onDraftChange({ defaultCurrency: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="settings-timezone" label="Timezone">
              <Input
                disabled={!canWrite || patching}
                id="settings-timezone"
                value={draft.timezone}
                onChange={(event) => onDraftChange({ timezone: event.target.value })}
              />
            </FilterField>
          </div>
        </FilterPanel>
      </PlatformSection>

      {paymentEnabled ? (
        <PlatformSection title="Payments">
          <FilterPanel className={FILTER_PANEL_FLAT}>
            <label className={cn('flex items-center', adminSpacing.gap.md, adminTypography.body)}>
              <Checkbox
                checked={draft.stripeEnabled}
                disabled={!canWrite || patching}
                onCheckedChange={(checked) =>
                  onDraftChange({ stripeEnabled: checked === true })
                }
              />
              <span>Enable Stripe checkout</span>
            </label>
            <div className={cn('grid sm:grid-cols-2', adminSpacing.gap.xl)}>
              <FilterField htmlFor="settings-stripe-success-url" label="Checkout success URL">
                <Input
                  disabled={!canWrite || patching}
                  id="settings-stripe-success-url"
                  value={draft.stripeCheckoutSuccessUrl}
                  onChange={(event) =>
                    onDraftChange({ stripeCheckoutSuccessUrl: event.target.value })
                  }
                />
              </FilterField>
              <FilterField htmlFor="settings-stripe-cancel-url" label="Checkout cancel URL">
                <Input
                  disabled={!canWrite || patching}
                  id="settings-stripe-cancel-url"
                  value={draft.stripeCheckoutCancelUrl}
                  onChange={(event) =>
                    onDraftChange({ stripeCheckoutCancelUrl: event.target.value })
                  }
                />
              </FilterField>
              <FilterField htmlFor="settings-stripe-secret-key" label="Secret key">
                <PasswordInput
                  disabled={!canWrite || patching}
                  id="settings-stripe-secret-key"
                  placeholder={maskedSecrets?.stripe_secret_key?.trim() || 'Leave empty to keep current'}
                  value={draft.stripeSecretKey}
                  onChange={(event) => onDraftChange({ stripeSecretKey: event.target.value })}
                />
              </FilterField>
              <FilterField htmlFor="settings-stripe-webhook-secret" label="Webhook secret">
                <PasswordInput
                  disabled={!canWrite || patching}
                  id="settings-stripe-webhook-secret"
                  placeholder={
                    maskedSecrets?.stripe_webhook_secret?.trim() || 'Leave empty to keep current'
                  }
                  value={draft.stripeWebhookSecret}
                  onChange={(event) => onDraftChange({ stripeWebhookSecret: event.target.value })}
                />
              </FilterField>
            </div>
          </FilterPanel>
        </PlatformSection>
      ) : null}

      <div className={cn('flex flex-wrap items-center', adminSpacing.gap.md)}>
        <PrimaryActionButton
          disabled={saveDisabled}
          loading={patching}
          type="button"
          onClick={onSave}
        >
          Save changes
        </PrimaryActionButton>
        <SecondaryActionButton
          disabled={!canWrite || !draftDirty || patching}
          type="button"
          onClick={onDiscard}
        >
          Discard changes
        </SecondaryActionButton>
      </div>
    </div>
  );
}
