import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import type { TaxProfile } from '@/api/types';
import { CustomerDetailFieldRow } from '@/domains/customers/customer_detail_field_row';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { COMPACT_TOOLBAR_ROW_CLASS } from '@/shell/filter_panel';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { isValidationError } from '@/lib/admin_validation_error';
import { panelError } from '@/shell/panel_error';
import { ValidationErrorBlock } from '@/shell/validation_error_block';
import { adminTypography } from '@/lib/admin_kit';

export type CustomerDetailTaxTabProps = {
  taxProfile: TaxProfile | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftCountryCode: string;
  draftTaxRegion: string;
  draftTaxScheme: string;
  draftTaxRateBps: string;
  onDraftCountryCodeChange: (value: string) => void;
  onDraftTaxRegionChange: (value: string) => void;
  onDraftTaxSchemeChange: (value: string) => void;
  onDraftTaxRateBpsChange: (value: string) => void;
  saving: boolean;
  saveError: Error | undefined;
  saveSuccess: boolean;
  canSave: boolean;
  onSave: () => void;
};

export function CustomerDetailTaxTab({
  taxProfile,
  fetching,
  error,
  hasSnapshot,
  draftCountryCode,
  draftTaxRegion,
  draftTaxScheme,
  draftTaxRateBps,
  onDraftCountryCodeChange,
  onDraftTaxRegionChange,
  onDraftTaxSchemeChange,
  onDraftTaxRateBpsChange,
  saving,
  saveError,
  saveSuccess,
  canSave,
  onSave,
}: CustomerDetailTaxTabProps) {
  return (
    <CustomerTabShell
      blockingErrorTitle="Could not load tax profile"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      <section className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle className={adminTypography.sectionTitle}>Tax profile</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4">
            <form
              className="grid gap-4"
              onSubmit={(event) => {
                event.preventDefault();
                onSave();
              }}
            >
              <CustomerDetailPanel>
                <CustomerDetailFieldRow htmlFor="tax-country-code" label="Country code">
                  <Input
                    disabled={!canSave}
                    id="tax-country-code"
                    value={draftCountryCode}
                    onChange={(event) => onDraftCountryCodeChange(event.target.value)}
                  />
                </CustomerDetailFieldRow>
                <CustomerDetailFieldRow htmlFor="tax-region" label="Tax region">
                  <Input
                    disabled={!canSave}
                    id="tax-region"
                    value={draftTaxRegion}
                    onChange={(event) => onDraftTaxRegionChange(event.target.value)}
                  />
                </CustomerDetailFieldRow>
                <CustomerDetailFieldRow htmlFor="tax-scheme" label="Tax scheme">
                  <Input
                    disabled={!canSave}
                    id="tax-scheme"
                    value={draftTaxScheme}
                    onChange={(event) => onDraftTaxSchemeChange(event.target.value)}
                  />
                </CustomerDetailFieldRow>
                <CustomerDetailFieldRow htmlFor="tax-rate-bps" label="Tax rate (bps)">
                  <Input
                    disabled={!canSave}
                    id="tax-rate-bps"
                    inputMode="numeric"
                    type="number"
                    value={draftTaxRateBps}
                    onChange={(event) => onDraftTaxRateBpsChange(event.target.value)}
                  />
                </CustomerDetailFieldRow>
                {taxProfile?.customer_id ? (
                  <CustomerDetailRow label="Customer ID" value={taxProfile.customer_id} />
                ) : null}
              </CustomerDetailPanel>
              <div className={COMPACT_TOOLBAR_ROW_CLASS}>
                <PrimaryActionButton disabled={!canSave} loading={saving} type="submit">
                  Save
                </PrimaryActionButton>
              </div>
            </form>
          </CardContent>
        </Card>

        {saveError ? (
          isValidationError(saveError) ? (
            <ValidationErrorBlock error={saveError} title="Check tax profile fields" />
          ) : (
            panelError(saveError, 'Save failed')
          )
        ) : null}
        {saveSuccess ? (
          <p className={adminTypography.bodyMuted} role="status">
            Tax profile saved.
          </p>
        ) : null}
      </section>
    </CustomerTabShell>
  );
}
