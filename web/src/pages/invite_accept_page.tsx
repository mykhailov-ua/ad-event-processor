import { type FormEvent, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

import { publicAcceptInvite } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
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
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Invite accept failed';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Accept invite</CardTitle>
          <CardDescription>
            Set your password to join the team. Already have access?{' '}
            <Link className="text-foreground underline" to="/login">
              Sign in
            </Link>
            .
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          {error ? <ErrorBlock title="Invite accept failed" message={error} /> : null}
          {!inviteToken ? (
            <ErrorBlock
              title="Invite link invalid"
              message="Open the invite URL from your email. It must include ?token=..."
            />
          ) : null}
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <Label htmlFor="invite-password">Password</Label>
              <Input
                id="invite-password"
                type="password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="invite-confirm">Confirm password</Label>
              <Input
                id="invite-confirm"
                type="password"
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
    </div>
  );
}
