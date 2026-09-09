import { type FormEvent, useState } from 'react';
import { Check } from 'lucide-react';
import { Link } from 'react-router-dom';

import { publicActivate } from '@/api/auth_api';
import { Button } from '@/components/ui/button';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { AuthPageLayout } from '@/shell/auth_page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useMeta } from '@/hooks/use_meta';
import { normalizeSubmitError } from '@/lib/admin_error';
import { requireNonEmpty } from '@/lib/admin_validation_error';
import { productDisplayName } from '@/lib/product_display_name';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { uiMessageSurfaceClass } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';
import { PageSkeleton } from '@/shell/page_skeleton';
export function ActivatePage() {
  const { bootstrapComplete, loading } = useMeta();
  const [licenseToken, setLicenseToken] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [teamName, setTeamName] = useState('');
  const [error, setError] = useState<Error | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(undefined);

    const licenseResult = requireNonEmpty(licenseToken, 'License key', 'license_token');
    if (!licenseResult.ok) {
      setError(licenseResult.error);
      return;
    }
    const emailResult = requireNonEmpty(email, 'Email', 'email');
    if (!emailResult.ok) {
      setError(emailResult.error);
      return;
    }
    const passwordResult = requireNonEmpty(password, 'Password', 'password');
    if (!passwordResult.ok) {
      setError(passwordResult.error);
      return;
    }
    const teamResult = requireNonEmpty(teamName, 'Team name', 'team_name');
    if (!teamResult.ok) {
      setError(teamResult.error);
      return;
    }

    setSubmitting(true);
    try {
      await publicActivate({
        license_token: licenseResult.value,
        email: emailResult.value,
        password: passwordResult.value,
        team_name: teamResult.value,
      });
      window.location.replace('/');
    } catch (err: unknown) {
      setError(normalizeSubmitError(err, 'Activation failed'));
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return <PageSkeleton />;
  }

  if (bootstrapComplete) {
    return (
      <AuthPageLayout>
        <Card>
          <CardHeader>
            <CardTitle>License already activated</CardTitle>
            <CardDescription>
              {productDisplayName} is set up on this host. Sign in with your operator account.
            </CardDescription>
          </CardHeader>
          <CardContent className={cn('grid', adminSpacing.gap.xl)}>
            <div className={uiMessageSurfaceClass('success')} role="status">
              <div className={cn('flex items-center', adminSpacing.gap.md)}>
                <Check
                  aria-hidden
                  className="h-4 w-4 shrink-0 text-admin-positive"
                  strokeWidth={2.5}
                />
                <p className={cn(adminTypography.sectionTitle, 'text-admin-positive')}>
                  Activation complete
                </p>
              </div>
              <p>
                The license for this installation has already been applied. Use the email and
                password you created during activation.
              </p>
            </div>
            <Button asChild className="w-full" type="button" variant="brand">
              <Link to="/login">Go to sign in</Link>
            </Button>
          </CardContent>
        </Card>
      </AuthPageLayout>
    );
  }

  return (
    <AuthPageLayout>
      <Card>
        <CardHeader>
          <CardTitle>Activate your deployment</CardTitle>
          <CardDescription>
            Create the owner account and paste the license key from your vendor. One step, then you
            are signed in.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error ? <ErrorBlock title="Activation failed" error={error} /> : null}
          <form className={cn('grid', adminSpacing.gap.xl)} onSubmit={handleSubmit}>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="activate-license">License key</Label>
              <Textarea
                id="activate-license"
                className="min-h-[5rem] w-full"
                placeholder="Paste JWT from vendor email"
                required
                value={licenseToken}
                onChange={(event) => setLicenseToken(event.target.value)}
              />
            </div>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="activate-email">Your email</Label>
              <Input
                id="activate-email"
                type="email"
                autoComplete="username"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </div>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="activate-password">Password</Label>
              <PasswordInput
                id="activate-password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="activate-team">Team / company name</Label>
              <Input
                id="activate-team"
                required
                value={teamName}
                onChange={(event) => setTeamName(event.target.value)}
              />
            </div>
            <PrimaryActionButton className="w-full" loading={submitting} type="submit">
              Activate and sign in
            </PrimaryActionButton>
          </form>
        </CardContent>
      </Card>
    </AuthPageLayout>
  );
}
