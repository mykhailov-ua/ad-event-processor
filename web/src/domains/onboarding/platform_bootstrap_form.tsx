import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { bootstrapPlatformSettings } from '@/api/settings_api';
import type { PlatformBootstrapRequest } from '@/api/types';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { COLD_PATH_MAX_BODY_CHARS } from '@/lib/body_limits';
import { requireJsonObject, requireNonEmpty } from '@/lib/admin_validation_error';
import { toError } from '@/lib/admin_error';

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
    const tokenResult = requireNonEmpty(installToken, 'Setup token', 'install_token');
    if (!tokenResult.ok) {
      setError(tokenResult.error);
      return;
    }
    const jsonResult = requireJsonObject(bootstrapJson, 'Setup configuration', 'bootstrap_json');
    if (!jsonResult.ok) {
      setError(jsonResult.error);
      return;
    }
    setSubmitting(true);
    setError(undefined);
    setSuccess(false);
    try {
      await bootstrapPlatformSettings(
        tokenResult.value,
        jsonResult.value as PlatformBootstrapRequest
      );
      setSuccess(true);
      toast.success('Platform setup complete');
      setInstallToken('');
      onComplete?.();
    } catch (err: unknown) {
      setError(toError(err));
    } finally {
      setSubmitting(false);
    }
  }, [bootstrapJson, installToken, onComplete]);

  return (
    <div>
      <div className="grid gap-4" >
        <Label htmlFor="setup-install-token">Setup token</Label>
        <PasswordInput
          id="setup-install-token"
          autoComplete="off"
          value={installToken}
          onChange={(event) => setInstallToken(event.target.value)}
        />
      </div>
      <div>
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
         
          disabled={!installToken.trim() || !bootstrapJson.trim()}
          loading={submitting}
          onClick={() => void onSubmit()}
          type="button"
        >
          Complete setup
        </PrimaryActionButton>
      </div>
      {success ? (
        <p>
          Setup complete. Sign in with the admin account you configured.
        </p>
      ) : null}
      {error ? <ErrorBlock title="Setup failed" error={error} /> : null}
    </div>
  );
}
