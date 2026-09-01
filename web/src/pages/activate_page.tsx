import { type FormEvent, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';

import { publicActivate } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useMeta } from '@/hooks/use_meta';
import { PageSkeleton } from '@/components/system/page_skeleton';

export function ActivatePage() {
  const { bootstrapComplete, loading } = useMeta();
  const [licenseToken, setLicenseToken] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [teamName, setTeamName] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(undefined);
    setSubmitting(true);
    try {
      await publicActivate({
        license_token: licenseToken.trim(),
        email: email.trim(),
        password,
        team_name: teamName.trim(),
      });
      window.location.replace('/');
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Activation failed';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return <PageSkeleton />;
  }

  if (bootstrapComplete) {
    return <Navigate replace to="/login" />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>Activate deployment</CardTitle>
          <CardDescription>
            Create the owner account and apply your license in one step. Alternative to JSON setup on{' '}
            <Link className="text-foreground underline" to="/setup">
              Initial setup
            </Link>
            .
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          {error ? <ErrorBlock title="Activation failed" message={error} /> : null}
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <Label htmlFor="activate-license">License JWT</Label>
              <Textarea
                id="activate-license"
                className="font-mono text-xs"
                required
                value={licenseToken}
                onChange={(event) => setLicenseToken(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="activate-email">Owner email</Label>
              <Input
                id="activate-email"
                type="email"
                autoComplete="username"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="activate-password">Password</Label>
              <Input
                id="activate-password"
                type="password"
                autoComplete="new-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="activate-team">Team name</Label>
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
    </div>
  );
}
