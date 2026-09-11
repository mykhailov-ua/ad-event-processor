import type { ReactNode } from 'react';

export type UnstyledFormRowProps = {
  htmlFor?: string;
  label: string;
  hint?: string;
  children: ReactNode;
  'data-testid'?: string;
};

export function UnstyledFormRow({
  htmlFor,
  label,
  hint,
  children,
  'data-testid': testId,
}: UnstyledFormRowProps) {
  return (
    <div data-role="form-row" data-testid={testId}>
      <label data-role="form-row-label" htmlFor={htmlFor}>
        {label}
      </label>
      {hint ? <span data-role="form-row-hint">{hint}</span> : null}
      <div data-role="form-row-control">{children}</div>
    </div>
  );
}
