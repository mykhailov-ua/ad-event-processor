import type { ReactNode } from 'react';

import { customerDetailPanelClass } from '@/domains/customers/customer_detail_classes';
import { cn } from '@/lib/utils';

export type CustomerDetailPanelProps = {
  children: ReactNode;
};

export function CustomerDetailPanel({ children, }: CustomerDetailPanelProps) {
  return <div >{children}</div>;
}
