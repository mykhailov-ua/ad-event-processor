import type { ReactNode } from 'react';

import {
  customerDetailRowClass,
  customerDetailRowLabelClass,
  customerDetailRowValueClass,
} from '@/domains/customers/customer_detail_classes';

export function CustomerDetailRow({
  label,
  value,
}: {
  label: string;
  value: string | number | undefined | ReactNode;
}) {
  const display =
    value == null || value === ''
      ? '-'
      : typeof value === 'string' || typeof value === 'number'
        ? String(value)
        : value;
  return (
    <div className={customerDetailRowClass} >
      <span className={customerDetailRowLabelClass} >{label}</span>
      <span className={customerDetailRowValueClass} >{display}</span>
    </div>
  );
}
