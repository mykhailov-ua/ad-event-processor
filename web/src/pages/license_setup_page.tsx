import { Link } from 'react-router-dom';

import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useMeta } from '@/hooks/use_meta';
import { licenseStateLabel } from '@/lib/install_meta';

export function LicenseSetupPage() {
  const { meta, refreshMeta } = useMeta();
  const stateLabel = licenseStateLabel(meta);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>Apply license</CardTitle>
          <CardDescription>
            A valid license JWT is required before using the operator console. Current state:{' '}
            {stateLabel}.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <LicenseApplyForm
            onApplied={() => {
              refreshMeta();
            }}
          />
          <p className="text-sm text-muted-foreground">
            License management remains available later under{' '}
            <Link className="text-foreground underline" to="/settings/license">
              Settings
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
