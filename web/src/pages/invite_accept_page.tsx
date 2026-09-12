import { type FormEvent, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

import { publicAcceptInvite } from '@/api/auth_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { AuthPageLayout } from '@/shell/auth_page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import { normalizeSubmitError } from '@/lib/admin_error';
import { requireNonEmpty } from '@/lib/admin_validation_error';
import { adminSpacing } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export function InviteAcceptPage() {
  const [searchParams] = useSearchParams();
  const inviteToken = useMemo(() => searchParams.get('token')?.trim() ?? '', [searchParams]);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [clientError, setClientError] = useState<string | undefined>();
  const [apiError, setApiError] = useState<Error | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setClientError(undefined);
    setApiError(undefined);
    if (!inviteToken) {
      setClientError('Invite token missing from URL query (?token=...)');
      return;
    }
    const passwordResult = requireNonEmpty(password, 'Password', 'password');
    if (!passwordResult.ok) {
      setClientError(passwordResult.error.message);
      return;
    }
    const confirmResult = requireNonEmpty(confirmPassword, 'Confirm password', 'confirm_password');
    if (!confirmResult.ok) {
      setClientError(confirmResult.error.message);
      return;
    }
    if (passwordResult.value !== confirmResult.value) {
      setClientError('Passwords do not match');
      return;
    }
    setSubmitting(true);
    try {
      await publicAcceptInvite({ token: inviteToken, password: passwordResult.value });
      window.location.replace('/');
    } catch (err: unknown) {
      setApiError(normalizeSubmitError(err, 'Invite accept failed'));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthPageLayout>
      <Card>
        <CardHeader>
          <CardTitle>Accept invite</CardTitle>
          <CardDescription>
            Set your password to join the team. Already have access?{' '}
            <Link className="text-primary hover:underline" to="/login">
              Sign in
            </Link>
            .
          </CardDescription>
        </CardHeader>
        <CardContent className={cn('grid', adminSpacing.gap.xl)}>
          {apiError ? <ErrorBlock title="Invite accept failed" error={apiError} /> : null}
          {clientError ? (
            <ErrorBlock title="Invite accept failed" message={clientError} />
          ) : null}
          {!inviteToken ? (
            <ErrorBlock
              title="Invite link invalid"
              message="Open the invite URL from your email. It must include ?token=..."
            />
          ) : null}
          <form className={cn('grid', adminSpacing.gap.xl)} onSubmit={handleSubmit}>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="invite-password">Password</Label>
              <PasswordInput
                id="invite-password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="invite-confirm">Confirm password</Label>
              <PasswordInput
                id="invite-confirm"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
              />
            </div>
            <PrimaryActionButton
              className="w-full"
              disabled={!inviteToken}
              loading={submitting}
              type="submit"
            >
              Accept invite
            </PrimaryActionButton>
          </form>
        </CardContent>
      </Card>
    </AuthPageLayout>
  );
}
