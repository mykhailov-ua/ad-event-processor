import { Component, type ErrorInfo, type ReactNode } from 'react';

import { AdminErrorPage } from '@/shell/admin_error_page';

type AppErrorBoundaryProps = {
  children: ReactNode;
  layout?: 'standalone' | 'embedded';
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
