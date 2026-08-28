import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type {
  IntegrationSchema,
  IntegrationTemplateCatalogEntry,
} from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { StubBanner } from '../system/stub_banner.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './templates.module.css';

export type TemplatesPanelProps = {
  catalog: IntegrationTemplateCatalogEntry[];
  imported: IntegrationSchema[];
  loading: boolean;
  error: unknown;
  stubUnavailable: boolean;
  busy: boolean;
  onImport: (names: string[]) => void;
};

export function TemplatesPanel({
  catalog,
  imported,
  loading,
  error,
  stubUnavailable,
  busy,
  onImport,
}: TemplatesPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);
  const [selected, setSelected] = useState<Record<string, boolean>>({});

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Templates access denied" />;
  }

  if (error && !stubUnavailable) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load template catalog" />;
  }

  const toggle = (name: string) => {
    setSelected((prev) => ({ ...prev, [name]: !prev[name] }));
  };

  const selectedNames = Object.entries(selected)
    .filter(([, on]) => on)
    .map(([name]) => name);

  return (
    <div className={shared.panelRoot} data-testid="integrations-templates-page">
      <PageChrome
        title="Template import"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />

      {stubUnavailable ? (
        <StubBanner
          title="Template catalog unavailable"
          message="Bundled template catalog is not configured on this control plane."
        />
      ) : null}

      <section>
        <h2 className={shared.sectionTitle}>Bundled catalog</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsCatalog}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              Name
            </span>
            <span className={shared.gridCell} role="columnheader">
              Version
            </span>
            <span className={shared.gridCell} role="columnheader">
              Category
            </span>
            <span className={shared.gridCell} role="columnheader">
              Kind
            </span>
            <span className={shared.gridCell} role="columnheader">
              Import
            </span>
          </div>
          {catalog.length === 0 && !loading ? (
            <EmptyState message="No bundled templates in catalog." />
          ) : (
            catalog.map((row) => {
              const name = row.name ?? '';
              return (
                <div key={name} className={`${shared.gridRow} ${styles.colsCatalog}`} role="row">
                  <span className={shared.gridCell} role="gridcell">
                    {name}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.version}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.category}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {row.kind}
                  </span>
                  <span className={shared.gridCell} role="gridcell">
                    {canWrite ? (
                      <label className={shared.checkboxRow}>
                        <input
                          type="checkbox"
                          checked={Boolean(selected[name])}
                          onChange={() => toggle(name)}
                          data-testid={`template-select-${name}`}
                        />
                        <span>Select</span>
                      </label>
                    ) : (
                      '-'
                    )}
                  </span>
                </div>
              );
            })
          )}
        </div>
        {canWrite ? (
          <div className={shared.toolbar}>
            <Button
              size="sm"
              variant="primary"
              disabled={busy || selectedNames.length === 0}
              data-testid="template-import-submit"
              onClick={() => onImport(selectedNames)}
            >
              Import selected
            </Button>
            <Button
              size="sm"
              variant="secondary"
              disabled={busy}
              onClick={() => onImport([])}
            >
              Import all
            </Button>
          </div>
        ) : null}
      </section>

      {imported.length > 0 ? (
        <section>
          <h2 className={shared.sectionTitle}>Last import result</h2>
          <ul>
            {imported.map((row) => (
              <li key={row.id}>
                {row.name} v{row.version} ({row.kind})
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
