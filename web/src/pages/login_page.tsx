import { type FormEvent, useState } from 'react';
import { Link, Navigate } from 'react-router-dom';

import { login } from '@/api/auth_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { AuthPageLayout } from '@/shell/auth_page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import { useMeta } from '@/hooks/use_meta';
import { normalizeSubmitError } from '@/lib/admin_error';
import { requireNonEmpty } from '@/lib/admin_validation_error';
import { productConsoleTagline } from '@/lib/product_display_name';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';
export function LoginPage() {
  const { bootstrapComplete, loading: metaLoading } = useMeta();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<Error | undefined>();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(undefined);

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

    setSubmitting(true);

    try {
      await login({ email: emailResult.value, password: passwordResult.value });
      window.location.replace('/');
    } catch (err: unknown) {
      setError(normalizeSubmitError(err, 'Sign in failed'));
    } finally {
      setSubmitting(false);
    }
  }

  if (metaLoading) {
    return <PageSkeleton />;
  }

  if (!bootstrapComplete) {
    return <Navigate replace to="/activate" />;
  }

  return (
    <AuthPageLayout>
      <Card>
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
          <CardDescription>{productConsoleTagline()}</CardDescription>
        </CardHeader>
        <CardContent>
          {error ? <ErrorBlock title="Sign in failed" error={error} /> : null}
          <form className={cn('grid', adminSpacing.gap.xl)} onSubmit={handleSubmit}>
            <div className={cn('grid', adminSpacing.gap.md)}>
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
            <div className={cn('grid', adminSpacing.gap.md)}>
              <Label htmlFor="password">Password</Label>
              <PasswordInput
                id="password"
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
          <p className={cn('text-center', adminTypography.bodyMuted)}>
            <Link className="text-primary hover:underline" to="/activate">
              Activate with license
            </Link>
          </p>
        </CardContent>
      </Card>
    </AuthPageLayout>
  );
}
