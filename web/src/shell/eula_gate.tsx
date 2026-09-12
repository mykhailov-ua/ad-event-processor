import type { ReactNode } from 'react';

import { logout } from '@/api/auth_api';
import { ApiError } from '@/api/client';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
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
    return <PageSkeleton />;
  }

  if (error && !(error instanceof ApiError && error.status === 404)) {
    return <ErrorBlock title="Could not load EULA status" error={error} />;
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
            <div className={cn('border border-border whitespace-pre-wrap', adminTypography.body)}>
              {eulaText?.trim() ? eulaText : 'EULA text unavailable from server.'}
            </div>

            {acceptError ? <ErrorBlock title="Accept failed" error={acceptError} /> : null}
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
              <div className={cn('grid w-full', adminTypography.bodyMuted, 'gap-3')}>
                <p>
                  Your session lacks settings:write permission required to accept the EULA. Sign out
                  and ask an administrator to accept, or use an account with settings access.
                </p>
                <SecondaryActionButton
                  type="button"
                  onClick={() => {
                    void logout()
                      .catch(() => undefined)
                      .finally(() => {
                        window.location.replace('/login');
                      });
                  }}
                >
                  Sign out
                </SecondaryActionButton>
              </div>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
