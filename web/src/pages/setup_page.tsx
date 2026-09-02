import { Link, Navigate } from 'react-router-dom';

import { PlatformBootstrapForm } from '@/domains/onboarding/platform_bootstrap_form';
import { AdminDevModeEntry } from '@/shell/admin_dev_mode_entry';
import { ErrorBlock } from '@/shell/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useMeta } from '@/hooks/use_meta';
import { PageSkeleton } from '@/shell/page_skeleton';

export function SetupPage() {
  const { bootstrapComplete, error, loading, refreshMeta } = useMeta();

  if (loading) {
    return <PageSkeleton />;
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background p-4">
        <ErrorBlock title="Could not load install status" message={error.message} />
        <div className="w-full max-w-sm">
          <AdminDevModeEntry />
        </div>
      </div>
    );
  }

  if (bootstrapComplete) {
    return <Navigate replace to="/login" />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>Initial setup</CardTitle>
          <CardDescription>
            First-run platform configuration for ad-event-processor. Requires the setup token from
            your deployment bundle.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <PlatformBootstrapForm
            onComplete={() => {
              refreshMeta();
            }}
          />
          <p className="mt-4 text-center text-sm text-muted-foreground">
            Have a license JWT already?{' '}
            <Link className="text-foreground underline" to="/activate">
              Activate with license
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
