import { SecondaryActionButton, PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { Textarea } from '@/components/ui/textarea';
import type { AccessCatalogResponse } from '@/api/access_api';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export type AccessRolesViewProps = {
  canRead: boolean;
  canWrite: boolean;
  catalog: AccessCatalogResponse | undefined;
  catalogError: Error | undefined;
  catalogFetching: boolean;
  yamlError: Error | undefined;
  yamlFetching: boolean;
  draftYaml: string;
  onDraftYamlChange: (value: string) => void;
  onValidate: () => void;
  onApply: () => void;
  validating: boolean;
  applying: boolean;
  validationError: Error | undefined;
  actionError: Error | undefined;
  syncDraftFromServer: () => void;
};

export function AccessRolesView({
  canRead,
  canWrite,
  catalog,
  catalogError,
  catalogFetching,
  yamlError,
  yamlFetching,
  draftYaml,
  onDraftYamlChange,
  onValidate,
  onApply,
  validating,
  applying,
  validationError,
  actionError,
  syncDraftFromServer,
}: AccessRolesViewProps) {
  if (!canRead) {
    return <ErrorBlock title="Forbidden" error={new Error('access:read required')} />;
  }

  return (
    <div className={cn('flex max-w-3xl flex-col', adminSpacing.gap.xl)}>
      {catalogError ? (
        <ErrorBlock title="Could not load capability catalog" error={catalogError} />
      ) : null}
      {yamlError ? <ErrorBlock title="Could not load roles YAML" error={yamlError} /> : null}
      {validationError ? <ErrorBlock title="Validation failed" error={validationError} /> : null}
      {actionError ? <ErrorBlock title="Apply failed" error={actionError} /> : null}

      <section className={cn('grid', adminSpacing.gap.md)}>
        <h2 className={adminTypography.sectionTitle}>Capability catalog</h2>
        {catalogFetching && !catalog ? (
          <p>Loading catalog...</p>
        ) : (
          <p className={adminTypography.bodyMuted}>
            {catalog?.capabilities.length ?? 0} capabilities, {catalog?.permissions.length ?? 0}{' '}
            permissions (server-owned).
          </p>
        )}
      </section>

      <section className={cn('grid', adminSpacing.gap.md)}>
        <h2 className={adminTypography.sectionTitle}>Roles YAML</h2>
        <Textarea
          aria-label="Roles YAML"
          className={cn('min-h-[24rem]', adminTypography.monoData)}
          disabled={!canWrite || yamlFetching}
          onChange={(event) => onDraftYamlChange(event.target.value)}
          value={draftYaml}
        />
        <div className={adminSpacing.flex.buttonGroup}>
          <SecondaryActionButton
            disabled={yamlFetching}
            onClick={syncDraftFromServer}
            type="button"
          >
            Reload from server
          </SecondaryActionButton>
          <SecondaryActionButton
            disabled={!canWrite || validating || !draftYaml.trim()}
            loading={validating}
            onClick={onValidate}
            type="button"
          >
            Validate
          </SecondaryActionButton>
          <PrimaryActionButton
            disabled={!canWrite || applying || !draftYaml.trim()}
            loading={applying}
            onClick={onApply}
            type="button"
          >
            Apply
          </PrimaryActionButton>
        </div>
      </section>
    </div>
  );
}
