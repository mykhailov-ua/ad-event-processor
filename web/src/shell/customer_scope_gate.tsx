import type { ReactNode } from 'react';

import { EmptyState } from '@/shell/empty_state';

export type CustomerScopeGateProps = {
  customerId: string;
  children: ReactNode;
  title?: string;
  description?: string;
  /** data-testid for scope prompt */
  testId?: string;
};

const DEFAULT_TITLE = 'Select a customer';
const DEFAULT_DESCRIPTION =
  'Enter a customer UUID in the Customer ID field above to load customer-scoped data.';

export function CustomerScopeGate({
  customerId,
  children,
  title,
  description,
  testId = 'customer-scope-gate',
}: CustomerScopeGateProps) {
  if (!customerId.trim()) {
    return (
      <div data-testid={testId}>
        <EmptyState
          description={description ?? DEFAULT_DESCRIPTION}
          title={title ?? DEFAULT_TITLE}
          variant="blank-slate"
        />
      </div>
    );
  }

  return children;
}
