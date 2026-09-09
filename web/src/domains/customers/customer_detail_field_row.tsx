import type { ReactNode } from 'react';

import { Label } from '@/components/ui/label';
import {
  customerDetailRowClass,
  customerDetailRowLabelClass,
} from '@/domains/customers/customer_detail_classes';

export type CustomerDetailFieldRowProps = {
  children: ReactNode;
  htmlFor: string;
  label: string;
};

export function CustomerDetailFieldRow({ children, htmlFor, label }: CustomerDetailFieldRowProps) {
  return (
    <div >
      <Label  htmlFor={htmlFor}>
        {label}
      </Label>
      <div >{children}</div>
    </div>
  );
}
