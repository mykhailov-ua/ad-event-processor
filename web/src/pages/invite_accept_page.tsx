import { type FormEvent, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

import { publicAcceptInvite } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';

export function InviteAcceptPage() {
  const [searchParams] = useSearchParams();
  const inviteToken = useMemo(() => searchParams.get('token')?.trim() ?? '', [searchParams]);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(undefined);
    if (!inviteToken) {
      setError('Invite token missing from URL query (?token=...)');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    setSubmitting(true);
    try {
      await publicAcceptInvite({ token: inviteToken, password });
      window.location.replace('/');
    } catch (err: unknown) {
      const message =
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : 'Invite accept failed';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div >
      <Card >
        <CardHeader>
          <CardTitle>Accept invite</CardTitle>
          <CardDescription>
            Set your password to join the team. Already have access?{' '}
            <Link  to="/login">
              Sign in
            </Link>
            .
          </CardDescription>
        </CardHeader>
        <CardContent >
          {error ? <ErrorBlock title="Invite accept failed" message={error} /> : null}
          {!inviteToken ? (
            <ErrorBlock
              title="Invite link invalid"
              message="Open the invite URL from your email. It must include ?token=..."
            />
          ) : null}
          <form  onSubmit={handleSubmit}>
            <div >
              <Label htmlFor="invite-password">Password</Label>
              <PasswordInput
                id="invite-password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div >
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
             
              disabled={!inviteToken}
              loading={submitting}
              type="submit"
            >
              Accept invite
            </PrimaryActionButton>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
