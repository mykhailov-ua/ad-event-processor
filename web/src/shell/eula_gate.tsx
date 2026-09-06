import type { ReactNode } from 'react';

import { ApiError } from '@/api/client';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { useEulaGate } from '@/shell/use_eula_gate';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

export type EulaGateProps = {
  children: ReactNode;
};

export function EulaGate({ children }: EulaGateProps) {
  const { loading, error, eulaText, accepting, acceptError, blocked, canAccept, onAccept } =
    useEulaGate();

  if (loading) {
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
          className="max-w-2xl p-0"
          onEscapeKeyDown={(event) => event.preventDefault()}
          onInteractOutside={(event) => event.preventDefault()}
        >
          <DialogHeader>
            <DialogTitle>End user license agreement</DialogTitle>
            <DialogDescription>
              Accept the current EULA version before continuing in the admin console.
            </DialogDescription>
          </DialogHeader>

          <DialogBody>
            <div className="rounded-md border border-border whitespace-pre-wrap text-sm">
              {eulaText?.trim() ? eulaText : 'EULA text unavailable from server.'}
            </div>

            {acceptError ? (
              <ErrorBlock title="Accept failed" message={acceptError.message} />
            ) : null}
          </DialogBody>

          <DialogFooter>
            {canAccept ? (
              <PrimaryActionButton
                disabled={accepting}
                loading={accepting}
                type="button"
                onClick={onAccept}
              >
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
