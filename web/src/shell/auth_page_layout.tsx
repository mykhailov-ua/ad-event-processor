import type { ReactNode } from 'react';

export function AuthPageLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4 sm:p-6">
      <div className="w-full max-w-sm">{children}</div>
    </div>
  );
}
