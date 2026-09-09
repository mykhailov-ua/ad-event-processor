import type { ReactNode } from 'react';

import { customerDetailPanelClass } from '@/domains/customers/customer_detail_classes';
import { cn } from '@/lib/utils';

export type CustomerDetailPanelProps = {
  children: ReactNode;
  className?: string;
};

export function CustomerDetailPanel({ children, className }: CustomerDetailPanelProps) {
  return <div className={cn(customerDetailPanelClass, className)}>{children}</div>;
}
