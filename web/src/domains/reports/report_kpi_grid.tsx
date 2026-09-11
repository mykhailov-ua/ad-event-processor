import type { ReactNode } from 'react';

import { adminTypography } from '@/lib/admin_kit';

export type ReportKpiItem = {
  label: string;
  value: ReactNode;
};

export function ReportKpiGrid({ items }: { items: ReportKpiItem[] }) {
  return (
    <div className="grid gap-3 grid-cols-[repeat(auto-fit,minmax(11rem,1fr))] sm:grid-cols-2 lg:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} className="rounded-md border p-3">
          <div className={adminTypography.bodyMuted}>{item.label}</div>
          <div className={adminTypography.sectionTitle}>{item.value}</div>
        </div>
      ))}
    </div>
  );
}
