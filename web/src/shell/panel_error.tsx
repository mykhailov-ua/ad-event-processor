import { Link } from 'react-router-dom';

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
  if (error instanceof ApiError && error.code === 'LIMIT_EXCEEDED') {
    return (
      <StubBanner
        title="License limit reached"
        message={
          <>
            {userErrorMessage(error)} Upgrade your plan in <Link to="/settings">Settings</Link> to
            raise cost-sync or deployment limits.
          </>
        }
      />
    );
  }
  if (error instanceof ApiError && (error.status === 403 || error.code === 'FEATURE_REQUIRED')) {
    return (
      <StubBanner
        title={
          options.forbiddenTitle ??
          (error.code === 'FEATURE_REQUIRED' ? 'License required' : `${title} forbidden`)
        }
        message={
          error.code === 'FEATURE_REQUIRED' ? (
            <>
              {userErrorMessage(error)} Open <Link to="/settings">Settings</Link> to review your
              license tier.
            </>
          ) : (
            userErrorMessage(error)
          )
        }
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
