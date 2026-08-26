import { useMemo, useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  createIntegrationSchema,
  INTEGRATION_SCHEMA_STARTERS,
  type IntegrationSchemaKind,
} from '../helpers/integration_api.js';
import type { IntegrationSchemaDTO, CreateIntegrationSchemaBody } from '../types/integration.js';
import { SectionCard } from './section_card.js';
import { Button } from './button.js';
import { FormField } from './form_field.js';

const KIND_OPTIONS: { value: IntegrationSchemaKind; label: string }[] = [
  { value: 'inbound_tokens', label: 'Inbound tokens (traffic source)' },
  { value: 'outbound_postback', label: 'Outbound postback (affiliate)' },
  {
    value: 'affiliate_receive_postback',
    label: 'Affiliate receive postback (panel URL)',
  },
  { value: 'status_mapping', label: 'Status mapping' },
];

export type IntegrationSchemaAuthorPanelProps = {
  canWrite: boolean;
  onCreated?: (schema: IntegrationSchemaDTO) => void;
};

export function IntegrationSchemaAuthorPanel({
  canWrite,
  onCreated,
}: IntegrationSchemaAuthorPanelProps) {
  const [name, setName] = useState('');
  const [version, setVersion] = useState('1');
  const [starterKind, setStarterKind] = useState<IntegrationSchemaKind>('outbound_postback');
  const [schemaText, setSchemaText] = useState(() =>
    JSON.stringify(INTEGRATION_SCHEMA_STARTERS.outbound_postback, null, 2)
  );
  const [busy, setBusy] = useState(false);

  const loadStarter = (kind: IntegrationSchemaKind) => {
    setStarterKind(kind);
    setSchemaText(JSON.stringify(INTEGRATION_SCHEMA_STARTERS[kind], null, 2));
  };

  const parsedPreview = useMemo(() => {
    try {
      return JSON.parse(schemaText) as Record<string, unknown>;
    } catch {
      return null;
    }
  }, [schemaText]);

  const submit = async () => {
    if (!canWrite || busy) return;
    const trimmedName = name.trim();
    const versionNum = Number(version);
    if (!trimmedName) {
      pushToastMessage({ title: 'Name required', message: 'Enter a unique schema name.' });
      return;
    }
    if (!Number.isInteger(versionNum) || versionNum <= 0) {
      pushToastMessage({
        title: 'Invalid version',
        message: 'Version must be a positive integer.',
      });
      return;
    }
    let schemaBody: unknown;
    try {
      schemaBody = JSON.parse(schemaText);
    } catch {
      pushToastMessage({ title: 'Invalid JSON', message: 'Schema body must be valid JSON.' });
      return;
    }
    if (!schemaBody || typeof schemaBody !== 'object' || Array.isArray(schemaBody)) {
      pushToastMessage({ title: 'Invalid schema', message: 'Schema body must be a JSON object.' });
      return;
    }

    setBusy(true);
    const [created, err] = await to(
      createIntegrationSchema({
        name: trimmedName,
        version: versionNum,
        schema: schemaBody as CreateIntegrationSchemaBody['schema'],
      })
    );
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Create failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: 'Schema created',
      message: `${created?.name ?? trimmedName} v${versionNum} (${created?.kind ?? 'schema'})`,
    });
    setName('');
    setVersion('1');
    onCreated?.(created!);
  };

  if (!canWrite) return null;

  return (
    <SectionCard
      icon="code"
      title="Author custom schema"
      desc="POST JSON to integration_schemas - kind is inferred from tokens, url_template, or status_map."
    >
      <div className="form-row" data-testid="integration-schema-author">
        <FormField label="Name" htmlFor="schema-author-name">
          <input
            id="schema-author-name"
            className="form-input"
            data-testid="schema-author-name"
            value={name}
            disabled={busy}
            placeholder="custom_affiliate_postback"
            onChange={(e) => setName(e.target.value)}
          />
        </FormField>
        <FormField label="Version" htmlFor="schema-author-version">
          <input
            id="schema-author-version"
            type="number"
            min={1}
            className="form-input"
            data-testid="schema-author-version"
            value={version}
            disabled={busy}
            onChange={(e) => setVersion(e.target.value)}
          />
        </FormField>
        <FormField label="Starter template" htmlFor="schema-author-starter">
          <select
            id="schema-author-starter"
            className="form-input"
            data-testid="schema-author-starter"
            value={starterKind}
            disabled={busy}
            onChange={(e) => loadStarter(e.target.value as IntegrationSchemaKind)}
          >
            {KIND_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </FormField>
      </div>
      <FormField
        label="Schema JSON"
        htmlFor="schema-author-body"
        hint="Inner document only (version + tokens, url_template, or status_map). YAML is accepted by the API when sent as parsed JSON."
      >
        <textarea
          id="schema-author-body"
          className="form-input font-mono text-sm"
          rows={12}
          data-testid="schema-author-body"
          value={schemaText}
          disabled={busy}
          onChange={(e) => setSchemaText(e.target.value)}
        />
      </FormField>
      {parsedPreview === null ? (
        <p className="text-muted text-sm" data-testid="schema-author-json-error">
          JSON parse error
        </p>
      ) : null}
      <Button
        label={busy ? 'Creating...' : 'Create schema'}
        variant="primary"
        size="sm"
        loading={busy}
        disabled={busy || parsedPreview === null}
        data-testid="schema-author-submit"
        onClick={() => void submit()}
      />
    </SectionCard>
  );
}
