import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { setToastHandler, type ToastMessage } from '../helpers/toast_ui.js';
import { Icon } from './icon.js';

const MAX_TOASTS = 3;
const DISMISS_MS = 5000;

type ToastItem = ToastMessage & { id: string };

/**
 * Global toast stack wired to pushToastMessage.
 */
export function ToastStack() {
  const [items, setItems] = useState<ToastItem[]>([]);

  useEffect(() => {
    setToastHandler((msg) => {
      const id = crypto.randomUUID();
      setItems((prev) => {
        const next = [...prev, { ...msg, id }];
        while (next.length > MAX_TOASTS) next.shift();
        return next;
      });
      window.setTimeout(() => {
        setItems((prev) => prev.filter((t) => t.id !== id));
      }, DISMISS_MS);
    });
    return () => setToastHandler(null);
  }, []);

  if (items.length === 0) return null;

  return createPortal(
    <div className="toast-stack" role="status">
      {items.map((item) => (
        <div key={item.id} className="toast">
          <Icon name="info" size={16} className="toast__icon" />
          <div className="toast__content">
            <div className="toast__title">{item.title}</div>
            <div className="toast__message">{item.message}</div>
          </div>
        </div>
      ))}
    </div>,
    document.body,
  );
}
