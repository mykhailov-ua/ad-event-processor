import type { ReactNode } from 'react';

import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

const SECTION_TABLE_SURFACE_CLASS = 'min-w-0';

export function StatPanel({
  children,
  title,
  meta,
}: {
  children: ReactNode;
  title: ReactNode;
  meta?: ReactNode;
}) {
  return (
    <div >
      <div >
        <h3 >{title}</h3>
        {meta}
      </div>
      <div >
        {children}
      </div>
    </div>
  );
}

export function StatRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div >
      <span >{label}</span>
      <span >{value}</span>
    </div>
  );
}

export function PanelSection({
  children,
  title,
  meta,
}: {
  children: ReactNode;
  title: ReactNode;
  meta?: ReactNode;
}) {
  return (
    <section >
      <div >
        <h3 >{title}</h3>
        {meta}
      </div>
      <div >{children}</div>
    </section>
  );
}
