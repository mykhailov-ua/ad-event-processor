import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import { visibleSettingsCards } from '../../helpers/settings_catalog.js';
import { EmptyState } from '../system/empty_state.js';
import { PageChrome } from '../system/page_chrome.js';
import styles from './settings_shared.module.css';

export type SettingsHubProps = {
  title?: string;
  showIntro?: boolean;
};

export function SettingsHub({ title = 'Settings', showIntro = true }: SettingsHubProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const cards = visibleSettingsCards(permissions);

  return (
    <section className={styles.content} aria-label="Settings navigation">
      {title ? <h2 className={styles.sectionTitle}>{title}</h2> : null}
      {showIntro ? (
        <p className={styles.intro}>
          Platform configuration, license, domains, billing disputes, and team surfaces.
        </p>
      ) : null}
      {cards.length === 0 ? (
        <EmptyState message="No settings surfaces available for this session." />
      ) : (
        <div className={styles.cardGrid} role="list">
          {cards.map((card) => (
            <Link
              key={card.id}
              to={card.route}
              className={styles.card}
              role="listitem"
              data-testid={`settings-card-${card.id}`}
            >
              <h3 className={styles.cardTitle}>{card.title}</h3>
              <p className={styles.cardDesc}>{card.description}</p>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

export function SettingsSubnav() {
  const permissions = auth.getUser()?.permissions ?? [];
  const cards = visibleSettingsCards(permissions);
  if (cards.length === 0) return null;

  return (
    <nav className={styles.toolbar} aria-label="Settings subpages">
      {cards.map((card) => (
        <Link key={card.id} to={card.route} className={styles.bannerLink}>
          {card.title}
        </Link>
      ))}
    </nav>
  );
}
