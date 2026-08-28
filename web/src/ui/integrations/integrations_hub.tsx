import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import { visibleIntegrationCards } from '../../helpers/integrations_catalog.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { EmptyState } from '../system/empty_state.js';
import styles from './integrations_shared.module.css';

export function IntegrationsHub() {
  const permissions = auth.getUser()?.permissions ?? [];
  const cards = visibleIntegrationCards(permissions);

  if (!auth.getUser()) {
    return <ErrorBlock error={new Error('unauthorized')} fallbackTitle="Session required" />;
  }

  return (
    <div className={styles.root} data-testid="integrations-hub-page">
      <PageChrome title="Integrations" badge={<span>{cards.length} surfaces</span>} />
      <p className={styles.intro}>
        Connect ad networks, postbacks, supply files, and automation rules. Cards match your
        session permissions.
      </p>
      {cards.length === 0 ? (
        <EmptyState message="No integration surfaces available for this session." />
      ) : (
        <div className={styles.grid} role="list">
          {cards.map((card) => (
            <Link
              key={card.id}
              to={card.route}
              className={styles.card}
              role="listitem"
              data-testid={`integration-card-${card.id}`}
            >
              <h2 className={styles.cardTitle}>{card.title}</h2>
              <p className={styles.cardDesc}>{card.description}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
