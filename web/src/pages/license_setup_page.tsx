import { Link } from 'react-router-dom';

import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import { useLicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useMeta } from '@/hooks/use_meta';
import { licenseStateLabel } from '@/lib/install_meta';

export function LicenseSetupPage() {
  const { meta, refreshMeta } = useMeta();
  const stateLabel = licenseStateLabel(meta);
  const licenseLoad = useLicenseApplyFormLoad(true);

  return (
    <div >
      <Card >
        <CardHeader>
          <CardTitle>Apply license</CardTitle>
          <CardDescription>
            A valid license JWT is required before using the operator console. Current state:{' '}
            {stateLabel}.
          </CardDescription>
        </CardHeader>
        <CardContent >
          <LicenseApplyForm
            load={licenseLoad}
            onApplied={() => {
              refreshMeta();
            }}
          />
          <p >
            License management remains available later under{' '}
            <Link to="/settings">
              Settings
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
