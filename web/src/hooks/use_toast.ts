import { useCallback } from 'react';
import { pushToastMessage } from '../helpers/toast_ui.js';

/**
 * Enqueue toasts via the imperative toast stack installed in AppShell.
 */
export function useToast() {
  return useCallback((title: string, message: string, code?: string) => {
    pushToastMessage({ title, message, code });
  }, []);
}
