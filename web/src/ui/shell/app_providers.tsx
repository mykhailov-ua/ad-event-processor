import { useEffect, useState, type ReactNode } from 'react';
import { setConfirmHandler, type ConfirmRequest } from '../../helpers/confirm_ui.js';
import { setToastHandler, type ToastMessage } from '../../helpers/toast_ui.js';
import styles from './app_providers.module.css';

export function AppProviders({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<ToastMessage | null>(null);

  useEffect(() => {
    setToastHandler((msg: ToastMessage) => setToast(msg));
    setConfirmHandler(async (req: ConfirmRequest) => {
      if (req.entry.level === 'none') return true;
      const label = req.title ?? req.entry.label ?? 'Confirm action?';
      return window.confirm(label);
    });
    return () => {
      setToastHandler(null);
      setConfirmHandler(null);
    };
  }, []);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(null), 4000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  return (
    <>
      {children}
      {toast ? (
        <div className={styles.toast} role="status">
          <strong>{toast.title}</strong>
          {toast.message ? <span>{toast.message}</span> : null}
        </div>
      ) : null}
    </>
  );
}
