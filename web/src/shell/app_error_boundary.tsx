import { Component, type ErrorInfo, type ReactNode } from 'react';
import { useLocation } from 'react-router-dom';

import { AdminErrorPage } from '@/shell/admin_error_page';

// Catches render-throw in route subtree; does not intercept async fetch errors (those use ErrorBlock).
type AppErrorBoundaryProps = {
  children: ReactNode;
  layout?: 'standalone' | 'embedded';
  resetKey?: string;
};

type AppErrorBoundaryState = {
  error: Error | undefined;
  componentStack: string | undefined;
};

export class AppErrorBoundary extends Component<AppErrorBoundaryProps, AppErrorBoundaryState> {
  state: AppErrorBoundaryState = {
    error: undefined,
    componentStack: undefined,
  };

  static getDerivedStateFromError(error: Error): Partial<AppErrorBoundaryState> {
    return { error };
  }

  componentDidUpdate(prevProps: AppErrorBoundaryProps): void {
    if (prevProps.resetKey !== this.props.resetKey && this.state.error) {
      this.setState({ error: undefined, componentStack: undefined });
    }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    this.setState({ componentStack: info.componentStack ?? undefined });
    console.error('admin render error', error, info);
  }

  private handleRetry = (): void => {
    this.setState({ error: undefined, componentStack: undefined });
  };

  render() {
    const { error, componentStack } = this.state;
    if (error) {
      return (
        <AdminErrorPage
          componentStack={componentStack}
          error={error}
          kind="render"
          layout={this.props.layout ?? 'embedded'}
          onRetry={this.handleRetry}
        />
      );
    }

    return this.props.children;
  }
}

export function AppRouteErrorBoundary({
  children,
  layout = 'embedded',
}: {
  children: ReactNode;
  layout?: 'standalone' | 'embedded';
}) {
  const location = useLocation();
  const resetKey = `${location.pathname}${location.search}${location.key}`;
  return (
    <AppErrorBoundary layout={layout} resetKey={resetKey}>
      {children}
    </AppErrorBoundary>
  );
}
