import { Link } from 'react-router-dom';

import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import { useLicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useMeta } from '@/hooks/use_meta';
import { licenseStateLabel } from '@/lib/install_meta';
import { adminTypography } from '@/lib/admin_kit';

export function LicenseSetupPage() {
  const { meta, refreshMeta } = useMeta();
  const stateLabel = licenseStateLabel(meta);
  const licenseLoad = useLicenseApplyFormLoad(true);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4" >
      <Card className="w-full max-w-lg" >
        <CardHeader>
          <CardTitle>Apply license</CardTitle>
          <CardDescription>
            A valid license JWT is required before using the operator console. Current state:{' '}
            {stateLabel}.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4" >
          <LicenseApplyForm
            load={licenseLoad}
            onApplied={() => {
              refreshMeta();
            }}
          />
          <p className={adminTypography.bodyMuted} >
            License management remains available later under{' '}
            <Link className="text-foreground underline" to="/settings">
              Settings
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
