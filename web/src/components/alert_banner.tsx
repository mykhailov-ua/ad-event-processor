import { useState } from 'react';
import { dismissAlert, isAlertDismissed } from '../helpers/alert_dismiss.js';
import { Icon } from './icon.js';

export type AlertBannerProps = {
  variant?: 'warning' | 'error' | 'info';
  message: string;
  dismissKey?: string;
  onDismiss?: () => void;
};

export function AlertBanner({
  variant = 'info',
  message,
  dismissKey,
  onDismiss,
}: AlertBannerProps) {
  const [dismissed, setDismissed] = useState(() =>
    dismissKey ? isAlertDismissed(dismissKey) : false
  );

  if (dismissed) return null;

  const iconName =
    variant === 'error' ? 'alert-circle' : variant === 'warning' ? 'alert-triangle' : 'info';

  return (
    <div
      className={`alert-banner alert-banner--${variant} mb-4`}
      role={variant === 'error' ? 'alert' : 'status'}
    >
      <Icon name={iconName} size={16} className="alert-banner__icon" />
      <span className="alert-banner__text">{message}</span>
      {dismissKey ? (
        <button
          type="button"
          className="alert-banner__close"
          aria-label="Dismiss"
          onClick={() => {
            if (dismissKey) dismissAlert(dismissKey);
            onDismiss?.();
            setDismissed(true);
          }}
        >
          <Icon name="x" size={14} />
        </button>
      ) : null}
    </div>
  );
}
