// Client-side validation and action-guard errors (C1/C2).
// AdminValidationError messages are operator-safe and always surfaced in UI.
// Server 400 copy still comes from ApiError via userErrorMessage.
import { toast } from 'sonner';

import { ApiError } from '../api/api_error.ts';

export type ValidationErrorKind = 'input' | 'action' | 'format';

export class AdminValidationError extends Error {
  readonly kind: ValidationErrorKind;
  readonly field?: string;
  readonly code?: string;

  constructor(
    message: string,
    options: { kind?: ValidationErrorKind; field?: string; code?: string } = {}
  ) {
    super(message);
    this.name = 'AdminValidationError';
    this.kind = options.kind ?? 'input';
    if (options.field !== undefined) {
      this.field = options.field;
    }
    if (options.code !== undefined) {
      this.code = options.code;
    }
  }
}

export function validationError(
  message: string,
  options?: { kind?: ValidationErrorKind; field?: string; code?: string }
): AdminValidationError {
  return new AdminValidationError(message, options);
}

export function actionGuardError(message: string): AdminValidationError {
  return new AdminValidationError(message, { kind: 'action', code: 'ACTION_GUARD' });
}

export function isValidationError(error: unknown): error is AdminValidationError {
  return error instanceof AdminValidationError;
}

export function validationErrorMessage(
  error: unknown,
  fallback = 'Check the highlighted fields and try again.'
): string {
  if (isValidationError(error)) {
    return error.message;
  }
  if (error instanceof ApiError && error.status === 400 && error.message.trim() !== '') {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  if (typeof error === 'string' && error.trim() !== '') {
    return error;
  }
  return fallback;
}

export function toValidationError(err: unknown, fallback: string): AdminValidationError {
  if (isValidationError(err)) {
    return err;
  }
  if (err instanceof Error && err.message.trim() !== '') {
    return validationError(err.message);
  }
  return validationError(fallback);
}

export type ValidationOk<T> = { ok: true; value: T };
export type ValidationFail = { ok: false; error: AdminValidationError };
export type ValidationResult<T> = ValidationOk<T> | ValidationFail;

function fail(message: string, field?: string): ValidationFail {
  return { ok: false, error: validationError(message, field ? { field } : undefined) };
}

export function requireNonEmpty(value: string, label: string, field?: string): ValidationResult<string> {
  const trimmed = value.trim();
  if (!trimmed) {
    return fail(`${label} is required.`, field);
  }
  return { ok: true, value: trimmed };
}

export function requireInteger(
  value: string,
  label: string,
  options: { min?: number; max?: number; field?: string } = {}
): ValidationResult<number> {
  const trimmed = value.trim();
  if (!trimmed) {
    return fail(`${label} is required.`, options.field);
  }
  if (!/^-?\d+$/.test(trimmed)) {
    return fail(`${label} must be an integer.`, options.field);
  }
  const parsed = Number(trimmed);
  if (!Number.isSafeInteger(parsed)) {
    return fail(`${label} is out of range.`, options.field);
  }
  if (options.min != null && parsed < options.min) {
    return fail(`${label} must be at least ${options.min}.`, options.field);
  }
  if (options.max != null && parsed > options.max) {
    return fail(`${label} must be at most ${options.max}.`, options.field);
  }
  return { ok: true, value: parsed };
}

export function requirePositiveInteger(
  value: string,
  label: string,
  field?: string
): ValidationResult<number> {
  const parsed = requireInteger(value, label, { min: 1, field });
  if (!parsed.ok) {
    return parsed;
  }
  return parsed;
}

export function requireNonNegativeInteger(
  value: string,
  label: string,
  field?: string
): ValidationResult<number> {
  return requireInteger(value, label, { min: 0, field });
}

export function requirePositiveNumber(
  value: string,
  label: string,
  field?: string
): ValidationResult<number> {
  const trimmed = value.trim();
  if (!trimmed) {
    return fail(`${label} is required.`, field);
  }
  const parsed = Number(trimmed);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fail(`${label} must be a positive number.`, field);
  }
  return { ok: true, value: parsed };
}

export function requireHexLength(
  value: string,
  length: number,
  label: string,
  field?: string
): ValidationResult<string> {
  const trimmed = value.trim().toLowerCase();
  if (!trimmed) {
    return fail(`${label} is required.`, field);
  }
  const pattern = new RegExp(`^[0-9a-f]{${length}}$`);
  if (!pattern.test(trimmed)) {
    return fail(`${label} must be ${length} hexadecimal characters.`, field);
  }
  return { ok: true, value: trimmed };
}

export function requireZeroOrOne(value: string, label: string, field?: string): ValidationResult<number> {
  const parsed = requireInteger(value, label, { min: 0, max: 1, field });
  if (!parsed.ok) {
    return parsed;
  }
  return parsed;
}

export function requireJsonObject(value: string, label: string, field?: string): ValidationResult<Record<string, unknown>> {
  const trimmed = value.trim();
  if (!trimmed) {
    return fail(`${label} is required.`, field);
  }
  try {
    const parsed: unknown = JSON.parse(trimmed);
    if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return fail(`${label} must be a JSON object.`, field);
    }
    return { ok: true, value: parsed as Record<string, unknown> };
  } catch {
    return fail(`${label} must be valid JSON.`, field);
  }
}

export function requireJsonArray(value: string, label: string, field?: string): ValidationResult<unknown[]> {
  const trimmed = value.trim();
  if (!trimmed) {
    return fail(`${label} is required.`, field);
  }
  try {
    const parsed: unknown = JSON.parse(trimmed);
    if (!Array.isArray(parsed)) {
      return fail(`${label} must be a JSON array.`, field);
    }
    return { ok: true, value: parsed };
  } catch {
    return fail(`${label} must be valid JSON.`, field);
  }
}

export function requireDateRange(
  from: string,
  to: string,
  options: { fromLabel?: string; toLabel?: string } = {}
): ValidationResult<{ from: string; to: string }> {
  const fromLabel = options.fromLabel ?? 'From date';
  const toLabel = options.toLabel ?? 'To date';
  const fromTrimmed = from.trim();
  const toTrimmed = to.trim();
  if (!fromTrimmed) {
    return fail(`${fromLabel} is required.`, 'from');
  }
  if (!toTrimmed) {
    return fail(`${toLabel} is required.`, 'to');
  }
  const fromMs = Date.parse(fromTrimmed);
  const toMs = Date.parse(toTrimmed);
  if (Number.isNaN(fromMs)) {
    return fail(`${fromLabel} is invalid.`, 'from');
  }
  if (Number.isNaN(toMs)) {
    return fail(`${toLabel} is invalid.`, 'to');
  }
  if (fromMs >= toMs) {
    return fail(`${fromLabel} must be before ${toLabel}.`, 'from');
  }
  return { ok: true, value: { from: fromTrimmed, to: toTrimmed } };
}

export function requireAllNonEmpty(
  fields: { value: string; label: string; field?: string }[]
): ValidationResult<Record<string, string>> {
  const result: Record<string, string> = {};
  for (const entry of fields) {
    const checked = requireNonEmpty(entry.value, entry.label, entry.field);
    if (!checked.ok) {
      return checked;
    }
    result[entry.field ?? entry.label] = checked.value;
  }
  return { ok: true, value: result };
}

export function toastValidationError(error: unknown): void {
  toast.error(validationErrorMessage(error));
}
