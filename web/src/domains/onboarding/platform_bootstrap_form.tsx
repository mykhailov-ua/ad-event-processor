import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { bootstrapPlatformSettings } from '@/api/settings_api';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { COLD_PATH_MAX_BODY_CHARS } from '@/lib/body_limits';

const DEFAULT_BOOTSTRAP_JSON = `{
  "admin_email": "ops@example.com",
  "admin_password": "change-me",
  "config": {
    "tracking_domain": "track.example.com"
  }
}`;

export type PlatformBootstrapFormProps = {
  onComplete?: () => void;
};

export function PlatformBootstrapForm({ onComplete }: PlatformBootstrapFormProps) {
  const [installToken, setInstallToken] = useState('');
  const [bootstrapJson, setBootstrapJson] = useState(DEFAULT_BOOTSTRAP_JSON);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [success, setSuccess] = useState(false);

  const onSubmit = useCallback(async () => {
    const token = installToken.trim();
    const trimmed = bootstrapJson.trim();
    if (!token || !trimmed) {
      return;
    }
    setSubmitting(true);
    setError(undefined);
    setSuccess(false);
    try {
      const parsed: unknown = JSON.parse(trimmed);
      if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('Setup configuration must be a JSON object');
      }
      await bootstrapPlatformSettings(token, parsed as Record<string, unknown>);
      setSuccess(true);
      toast.success('Platform setup complete');
      setInstallToken('');
      onComplete?.();
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSubmitting(false);
    }
  }, [bootstrapJson, installToken, onComplete]);

  return (
    <div className="grid gap-4">
      <div className="grid gap-2">
        <Label htmlFor="setup-install-token">Setup token</Label>
        <Input
          id="setup-install-token"
          type="password"
          autoComplete="off"
          value={installToken}
          onChange={(event) => setInstallToken(event.target.value)}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="setup-bootstrap-json">Setup configuration</Label>
        <Textarea
          id="setup-bootstrap-json"
          value={bootstrapJson}
          maxLength={COLD_PATH_MAX_BODY_CHARS}
          onChange={(event) => setBootstrapJson(event.target.value)}
        />
      </div>
      <div>
        <PrimaryActionButton
          className="w-full"
          disabled={!installToken.trim() || !bootstrapJson.trim()}
          loading={submitting}
          onClick={() => void onSubmit()}
          type="button"
        >
          Complete setup
        </PrimaryActionButton>
      </div>
      {success ? (
        <p className="text-sm text-muted-foreground">Setup complete. Sign in with the admin account you configured.</p>
      ) : null}
      {error ? <ErrorBlock title="Setup failed" message={error.message} /> : null}
    </div>
  );
}
