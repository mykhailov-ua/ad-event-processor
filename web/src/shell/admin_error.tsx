import { panelError, type PanelErrorOptions } from '@/shell/panel_error';

export type AdminErrorVariant = 'blocking' | 'inline' | 'mutation';

export type AdminErrorProps = {
  error: Error;
  title: string;
  variant?: AdminErrorVariant;
  options?: PanelErrorOptions;
};

export function AdminError({ error, title, variant = 'inline', options }: AdminErrorProps) {
  void variant;
  return panelError(error, title, options);
}

export function AdminMutationError({ error, title, options }: Omit<AdminErrorProps, 'variant'>) {
  return <AdminError error={error} title={title} variant="mutation" options={options} />;
}
