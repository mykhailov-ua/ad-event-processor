import { cloneElement, isValidElement, useId, type ReactElement, type ReactNode } from 'react';

export type FormFieldProps = {
  label: string;
  htmlFor?: string;
  error?: string | null;
  hint?: string;
  reserveFeedback?: boolean;
  children: ReactElement;
};

/**
 * Labeled form field with hint/error feedback.
 */
export function FormField({
  label,
  htmlFor,
  error,
  hint,
  reserveFeedback,
  children,
}: FormFieldProps) {
  const autoId = useId();
  const controlId = htmlFor ?? autoId;

  const control = isValidElement<{ id?: string; 'aria-invalid'?: boolean }>(children)
    ? cloneElement(children, {
      id: children.props.id ?? controlId,
      'aria-invalid': error ? true : undefined,
    })
    : children;

  let feedback: ReactNode = null;
  if (error) {
    feedback = <div className="form-error" role="alert">{error}</div>;
  } else if (hint) {
    feedback = <p className="form-hint">{hint}</p>;
  }

  return (
    <div
      className={[
        'form-field',
        error ? 'form-field--error' : '',
        reserveFeedback ? 'form-field--reserve-feedback' : '',
      ].filter(Boolean).join(' ')}
    >
      <label className="form-label" htmlFor={controlId}>{label}</label>
      {control}
      <div className="form-field__feedback">{feedback}</div>
    </div>
  );
}
