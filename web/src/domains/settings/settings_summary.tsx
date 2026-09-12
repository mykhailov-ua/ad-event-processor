import { useState } from 'react';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';

import type { MetaResponse, PlatformSettingsView } from '@/api/types';
import { ingressSchemaLabel } from '@/domains/settings/settings_field_labels';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { copyTextToClipboard } from '@/lib/copy_text_to_clipboard';
import { userErrorMessage } from '@/lib/admin_error';
import { SecondaryActionButton } from '@/shell/action_buttons';
import { cn } from '@/lib/utils';

export type SettingsSummaryProps = {
  meta: MetaResponse | undefined;
  platformSnapshot: PlatformSettingsView | undefined;
};

function SummaryRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className={adminSpacing.stack.titleBlock}>
      <span className={adminTypography.labelMuted}>{label}</span>
      <span className={adminTypography.body}>{children}</span>
    </div>
  );
}

export function SettingsSummary({ meta, platformSnapshot }: SettingsSummaryProps) {
  const [copying, setCopying] = useState(false);
  const deploymentId = meta?.deployment_id?.trim() || '';
  const ingressSchemas = meta?.ingress_schemas ?? [];
  const clickTemplate = platformSnapshot?.click_url_template?.trim() ?? '';
  const openrtbTemplate = platformSnapshot?.openrtb_endpoint_template?.trim() ?? '';
  const trackingDomain = platformSnapshot?.config?.tracking_domain?.trim() ?? '';

  const onCopyDeploymentId = async () => {
    if (!deploymentId || copying) {
      return;
    }
    setCopying(true);
    try {
      await copyTextToClipboard(deploymentId);
      toast.success('Deployment ID copied');
    } catch (error) {
      toast.error(userErrorMessage(error));
    } finally {
      setCopying(false);
    }
  };

  return (
    <section className={cn('grid', adminSpacing.gap.lg)}>
      <h2 className={adminTypography.sectionTitle}>Deployment summary</h2>
      <div className={cn('grid sm:grid-cols-2', adminSpacing.gap.xl, adminTypography.body)}>
        <SummaryRow label="Product">
          {meta?.product_name?.trim() || 'ad-event-processor'}
        </SummaryRow>
        <SummaryRow label="Version">{meta?.version?.trim() || '-'}</SummaryRow>
        <SummaryRow label="Bootstrap">
          {(meta?.bootstrap_complete ?? platformSnapshot?.bootstrap_complete)
            ? 'Complete'
            : 'Pending'}
        </SummaryRow>
        <SummaryRow label="Payment module">
          {meta?.payment_enabled ? 'Enabled' : 'Disabled'}
        </SummaryRow>
        {deploymentId ? (
          <div className={adminSpacing.stack.titleBlock}>
            <span className={adminTypography.labelMuted}>Deployment ID</span>
            <div className={cn('flex flex-wrap items-center', adminSpacing.gap.md)}>
              <span className={adminTypography.monoData}>{deploymentId}</span>
              <SecondaryActionButton
                disabled={copying}
                loading={copying}
                type="button"
                onClick={() => void onCopyDeploymentId()}
              >
                Copy
              </SecondaryActionButton>
            </div>
          </div>
        ) : null}
        {ingressSchemas.length > 0 ? (
          <SummaryRow label="Ingress schemas">
            {ingressSchemas.map((schema) => ingressSchemaLabel(schema)).join(', ')}
          </SummaryRow>
        ) : null}
        <SummaryRow label="Click URL template">
          {clickTemplate || (trackingDomain ? '-' : 'Set tracking domain below')}
        </SummaryRow>
        <SummaryRow label="OpenRTB endpoint template">
          {openrtbTemplate || (trackingDomain ? '-' : 'Set tracking domain below')}
        </SummaryRow>
      </div>
      <p className={adminTypography.bodyMuted}>
        <Link className="text-foreground underline" to="/ops/domains">
          Manage tracking domains
        </Link>
      </p>
    </section>
  );
}
