import { ApiError } from '@/api/client';
import { userErrorMessage } from '@/lib/admin_error';
import { ErrorBlock } from '@/shell/error_block';
import { StubBanner } from '@/shell/stub_banner';

export type PanelErrorOptions = {
  forbiddenTitle?: string;
  unavailableTitle?: string;
};

export function isPanelStubError(error: Error | undefined): boolean {
  return error instanceof ApiError && (error.status === 403 || error.status === 501);
}

export function panelError(error: Error, title: string, options: PanelErrorOptions = {}) {
  if (
    error instanceof ApiError &&
    (error.code === 'PAYMENT_UNAVAILABLE' ||
      error.code === 'BILLING_UNAVAILABLE' ||
      error.code === 'FORECAST_UNAVAILABLE' ||
      error.code === 'CLICKHOUSE_UNAVAILABLE')
  ) {
    return (
      <StubBanner
        title={options.unavailableTitle ?? `${title} unavailable`}
        message={userErrorMessage(error)}
      />
    );
  }
  if (
    error instanceof ApiError &&
    (error.status === 403 || error.code === 'FEATURE_REQUIRED')
  ) {
    return (
      <StubBanner
        title={options.forbiddenTitle ?? (error.code === 'FEATURE_REQUIRED' ? 'License required' : `${title} forbidden`)}
        message={userErrorMessage(error)}
      />
    );
  }
  if (error instanceof ApiError && error.status === 501) {
    return (
      <StubBanner
        title={options.unavailableTitle ?? `${title} unavailable`}
        message={userErrorMessage(error)}
      />
    );
  }
  return <ErrorBlock title={title} error={error} />;
}
