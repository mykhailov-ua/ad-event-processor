import { type FormEvent, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';

import { login } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useMeta } from '@/hooks/use_meta';

export function LoginPage() {
  const { bootstrapComplete, loading: metaLoading } = useMeta();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(undefined);
    setSubmitting(true);

    try {
      await login({ email, password });
      window.location.replace('/');
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Sign in failed';
      setError(message);
    } finally {
      setSubmitting(false);
    }
  }

  if (metaLoading) {
    return <PageSkeleton />;
  }

  if (!bootstrapComplete) {
    return <Navigate replace to="/setup" />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
          <CardDescription>ad-event-processor operator console</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          {error ? <ErrorBlock title="Sign in failed" message={error} /> : null}
          <form className="grid gap-4" onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                autoComplete="username"
                required
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <PrimaryActionButton className="w-full" loading={submitting} type="submit">
              Sign in
            </PrimaryActionButton>
          </form>
          <p className="text-center text-sm text-muted-foreground">
            First install?{' '}
            <Link className="text-foreground underline" to="/setup">
              Run setup
            </Link>{' '}
            or{' '}
            <Link className="text-foreground underline" to="/activate">
              activate with license
            </Link>
            .
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
