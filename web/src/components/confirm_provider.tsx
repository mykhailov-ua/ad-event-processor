import { lazy, Suspense, useEffect, useRef, useState, type ReactNode } from 'react';
import { setConfirmHandler, type ConfirmRequest } from '../helpers/confirm_ui.js';

const ConfirmDialog = lazy(() =>
  import('./confirm_dialog.js').then((mod) => ({
    default: mod.ConfirmDialog,
  }))
);

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [request, setRequest] = useState<ConfirmRequest | null>(null);
  const resolveRef = useRef<((accepted: boolean) => void) | null>(null);

  useEffect(() => {
    setConfirmHandler(
      (req) =>
        new Promise<boolean>((resolve) => {
          resolveRef.current = resolve;
          setRequest(req);
        })
    );
    return () => {
      setConfirmHandler(() => Promise.resolve(true));
    };
  }, []);

  const settle = (accepted: boolean) => {
    resolveRef.current?.(accepted);
    resolveRef.current = null;
    setRequest(null);
  };

  return (
    <>
      {children}
      {request ? (
        <Suspense fallback={null}>
          <ConfirmDialog
            open
            level={request?.entry?.level ?? 'standard'}
            title={request?.title ?? request?.entry?.label}
            description={request?.description}
            onConfirm={() => settle(true)}
            onCancel={() => settle(false)}
          />
        </Suspense>
      ) : null}
    </>
  );
}
