import { useCallback, useEffect, useState, type ReactNode } from 'react';

import { acceptEula, getEulaStatus, getMeta } from '@/api/platform_api';
import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useSession } from '@/hooks/use_session';

export type EulaGateProps = {
  children: ReactNode;
};

export function EulaGate({ children }: EulaGateProps) {
  const {
    user,
    loading: sessionLoading,
    eulaRequired: bootstrapRequired,
    eulaAccepted: bootstrapAccepted,
    eulaVersion: bootstrapVersion,
  } = useSession();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [required, setRequired] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [version, setVersion] = useState<string | undefined>();
  const [eulaText, setEulaText] = useState<string | undefined>();
  const [accepting, setAccepting] = useState(false);
  const [acceptError, setAcceptError] = useState<Error | undefined>(undefined);

  const canAccept = user?.permissions?.includes('settings:write') ?? false;

  useEffect(() => {
    if (sessionLoading) {
      return;
    }
    const ctrl = new AbortController();

    async function loadMeta() {
      setLoading(true);
      setError(undefined);
      try {
        if (bootstrapRequired !== undefined) {
          const eulaRequired = bootstrapRequired === true;
          const eulaAccepted = bootstrapAccepted === true;
          setRequired(eulaRequired);
          setAccepted(eulaAccepted);
          setVersion(bootstrapVersion);

          if (eulaRequired && !eulaAccepted) {
            const fullStatus = await getEulaStatus(ctrl.signal);
            setVersion(fullStatus.version ?? bootstrapVersion);
            setEulaText(fullStatus.text);
          }
          return;
        }

        const meta = await getMeta(ctrl.signal);
        const eulaRequired = meta.eula_required === true;
        const eulaAccepted = meta.eula_accepted === true;
        setRequired(eulaRequired);
        setAccepted(eulaAccepted);
        setVersion(meta.eula_version);

        if (eulaRequired && !eulaAccepted) {
          const fullStatus = await getEulaStatus(ctrl.signal);
          setVersion(fullStatus.version ?? meta.eula_version);
          setEulaText(fullStatus.text);
        }
      } catch (err) {
        if (ctrl.signal.aborted) {
          return;
        }
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        if (!ctrl.signal.aborted) {
          setLoading(false);
        }
      }
    }

    void loadMeta();
    return () => ctrl.abort();
  }, [bootstrapAccepted, bootstrapRequired, bootstrapVersion, sessionLoading]);

  const blocked = !loading && !error && required && !accepted;

  const onAccept = useCallback(async () => {
    const acceptVersion = version?.trim();
    if (!acceptVersion) {
      setAcceptError(new Error('EULA version missing from server status'));
      return;
    }
    setAccepting(true);
    setAcceptError(undefined);
    try {
      const next = await acceptEula({ version: acceptVersion });
      setAccepted(next.accepted === true);
      setRequired(next.required === true);
      setVersion(next.version);
      if (next.text) {
        setEulaText(next.text);
      }
    } catch (err) {
      setAcceptError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setAccepting(false);
    }
  }, [version]);

  if (sessionLoading || loading) {
    return null;
  }

  if (error && !(error instanceof ApiError && error.status === 404)) {
    return <ErrorBlock title="Could not load EULA status" message={error.message} />;
  }

  return (
    <>
      {children}
      <Dialog open={blocked} onOpenChange={() => undefined}>
        <DialogContent
          className="max-w-2xl"
          onEscapeKeyDown={(event) => event.preventDefault()}
          onInteractOutside={(event) => event.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>End user license agreement</DialogTitle>
            <DialogDescription>
              Accept the current EULA version before continuing in the admin console.
            </DialogDescription>
          </DialogHeader>

          <div className="overflow-hidden rounded-md border border-border">
            <div className="ui-scrollbar max-h-96 overflow-y-auto whitespace-pre-wrap p-4 text-sm">
              {eulaText?.trim() ? eulaText : 'EULA text unavailable from server.'}
            </div>
          </div>

          {acceptError ? <ErrorBlock title="Accept failed" message={acceptError.message} /> : null}

          <DialogFooter>
            {canAccept ? (
            <PrimaryActionButton disabled={accepting} loading={accepting} onClick={() => void onAccept()} type="button">
              Accept EULA
            </PrimaryActionButton>
            ) : (
              <p className="text-sm text-muted-foreground">
                Your session lacks settings:write permission required to accept the EULA.
              </p>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
