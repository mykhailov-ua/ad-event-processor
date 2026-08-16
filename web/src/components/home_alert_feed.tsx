import { useState } from 'react';
import type { HomeAlertCard } from '../helpers/home_alerts.js';
import { dismissAlert, isAlertDismissed } from '../helpers/alert_dismiss.js';
import { Button, ButtonLink } from './button.js';
import { StatusBadge } from './status_badge.js';

export type HomeAlertFeedProps = {
  alerts: HomeAlertCard[];
};

/**
 * Session-dismissible home alert cards.
 */
export function HomeAlertFeed({ alerts }: HomeAlertFeedProps) {
  const [, bump] = useState(0);

  const visible = alerts.filter((a) => a?.id && !isAlertDismissed(`home.${a.id}`));
  if (visible.length === 0) return null;

  return (
    <section className="alert-feed section-block" data-testid="alert-feed">
      <h3 className="alert-feed__title">Alerts</h3>
      <ul className="alert-feed__list">
        {visible.map((alert) => {
          const tone = alert.level === 'critical' ? 'error' : 'warning';
          return (
            <li key={alert.id} className="alert-feed__item" data-testid={`alert-${alert.id}`}>
              <StatusBadge status={tone} label={alert.title} />
              <span>{alert.detail}</span>
              <div className="alert-feed__item-actions">
                {alert.route ? (
                  <ButtonLink href={alert.route} label="View" variant="ghost" size="sm" />
                ) : null}
                <Button
                  label="Dismiss"
                  variant="ghost"
                  size="sm"
                  data-testid={`alert-dismiss-${alert.id}`}
                  onClick={() => {
                    dismissAlert(`home.${alert.id}`);
                    bump((n) => n + 1);
                  }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
