import { ApiError } from '@/api/client';
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
  if (error instanceof ApiError && error.status === 403) {
    return (
      <StubBanner title={options.forbiddenTitle ?? `${title} forbidden`} message={error.message} />
    );
  }
  if (error instanceof ApiError && error.status === 501) {
    return (
      <StubBanner
        title={options.unavailableTitle ?? `${title} unavailable`}
        message={error.message}
      />
    );
  }
  return <ErrorBlock title={title} message={error.message} />;
}
