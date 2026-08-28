import { useState } from 'react';
import { Link } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import type { IntegrationSchema } from '../../helpers/integrations_api.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import shared from '../integrations/integrations_shared.module.css';
import styles from './schemas.module.css';

export type SchemasPanelProps = {
  items: IntegrationSchema[];
  selected: IntegrationSchema | null;
  loading: boolean;
  detailLoading: boolean;
  error: unknown;
  detailError: unknown;
  busy: boolean;
  onSelect: (id: string) => void;
  onCreate: (name: string, version: number, schemaText: string) => void;
  onApply: (schemaId: string, campaignId: string) => void;
};

export function SchemasPanel({
  items,
  selected,
  loading,
  detailLoading,
  error,
  detailError,
  busy,
  onSelect,
  onCreate,
  onApply,
}: SchemasPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'campaigns:write');
  const canList = canReadCampaigns(permissions);

  const [createName, setCreateName] = useState('');
  const [createVersion, setCreateVersion] = useState('1');
  const [createBody, setCreateBody] = useState('{\n  "kind": "inbound_tokens"\n}');
  const [applyCampaignId, setApplyCampaignId] = useState('');

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Schemas access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load schemas" />;
  }

  return (
    <div className={shared.panelRoot} data-testid="integrations-schemas-page">
      <PageChrome
        title="Integration schemas"
        badge={
          <Link to="/integrations" className={shared.bannerLink}>
            All integrations
          </Link>
        }
      />

      <section>
        <h2 className={shared.sectionTitle}>Stored schemas</h2>
        <div className={shared.gridTable} role="grid">
          <div className={`${shared.gridHeader} ${styles.colsList}`} role="row">
            <span className={shared.gridCell} role="columnheader">
              Name
            </span>
            <span className={shared.gridCell} role="columnheader">
              Version
            </span>
            <span className={shared.gridCell} role="columnheader">
              Kind
            </span>
            <span className={shared.gridCell} role="columnheader">
              Updated
            </span>
            <span className={shared.gridCell} role="columnheader">
              View
            </span>
          </div>
          {items.length === 0 && !loading ? (
            <EmptyState message="No integration schemas stored yet." />
          ) : (
            items.map((row) => (
              <div
                key={row.id}
                className={`${shared.gridRow} ${styles.colsList}`}
                role="row"
                data-testid={`schema-row-${row.id}`}
              >
                <span className={shared.gridCell} role="gridcell">
                  {row.name}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.version}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.kind}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  {row.updated_at ?? row.created_at ?? '-'}
                </span>
                <span className={shared.gridCell} role="gridcell">
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => row.id && onSelect(row.id)}
                  >
                    Open
                  </Button>
                </span>
              </div>
            ))
          )}
        </div>
      </section>

      {selected ? (
        <section>
          <h2 className={shared.sectionTitle}>
            {selected.name} v{selected.version}
          </h2>
          {detailError ? (
            <ErrorBlock error={detailError} fallbackTitle="Failed to load schema detail" />
          ) : detailLoading ? (
            <p className={shared.hint}>Loading schema...</p>
          ) : (
            <pre className={shared.codePreview}>
              {JSON.stringify(selected.schema, null, 2)}
            </pre>
          )}
          {canWrite ? (
            <div className={shared.formStack}>
              <label className={shared.field}>
                <span className={shared.fieldLabel}>Apply to campaign ID</span>
                <input
                  className={shared.textInput}
                  value={applyCampaignId}
                  onChange={(event) => setApplyCampaignId(event.target.value)}
                  data-testid="schema-apply-campaign-id"
                />
              </label>
              <Button
                size="sm"
                variant="primary"
                disabled={busy || !selected.id || !applyCampaignId.trim()}
                data-testid="schema-apply-submit"
                onClick={() => selected.id && onApply(selected.id, applyCampaignId.trim())}
              >
                Apply schema
              </Button>
            </div>
          ) : null}
        </section>
      ) : null}

      {canWrite ? (
        <section>
          <h2 className={shared.sectionTitle}>Create schema</h2>
          <div className={shared.formStack} data-testid="integration-schema-author">
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Name</span>
              <input
                className={shared.textInput}
                value={createName}
                onChange={(event) => setCreateName(event.target.value)}
                data-testid="schema-author-name"
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Version</span>
              <input
                className={shared.textInput}
                type="number"
                min={1}
                value={createVersion}
                onChange={(event) => setCreateVersion(event.target.value)}
                data-testid="schema-author-version"
              />
            </label>
            <label className={shared.field}>
              <span className={shared.fieldLabel}>Schema JSON</span>
              <textarea
                className={shared.textarea}
                value={createBody}
                onChange={(event) => setCreateBody(event.target.value)}
                data-testid="schema-author-body"
              />
            </label>
            <Button
              size="sm"
              variant="primary"
              disabled={busy}
              data-testid="schema-author-submit"
              onClick={() =>
                onCreate(createName.trim(), Number.parseInt(createVersion, 10) || 1, createBody)
              }
            >
              Create schema
            </Button>
          </div>
        </section>
      ) : null}
    </div>
  );
}
