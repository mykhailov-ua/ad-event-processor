import type { ReactNode } from 'react';

export type ReportKpiItem = {
  label: string;
  value: ReactNode;
};

export function ReportKpiGrid({ items }: { items: ReportKpiItem[] }) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {items.map((item) => (
        <div key={item.label} className="rounded-md border p-3">
          <div className="text-xs text-muted-foreground">{item.label}</div>
          <div className="text-lg font-medium">{item.value}</div>
        </div>
      ))}
    </div>
  );
}
