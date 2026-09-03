import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { shouldShowAdminErrorDetails } from '@/lib/admin_error';

type AdminErrorDetailsProps = {
  details: string;
};

export function AdminErrorDetails({ details }: AdminErrorDetailsProps) {
  const [copied, setCopied] = useState(false);

  if (!shouldShowAdminErrorDetails() || details.trim() === '') {
    return null;
  }

  async function copyDetails() {
    try {
      await navigator.clipboard.writeText(details);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="mt-4 rounded-md border border-zinc-200 dark:border-zinc-800">
      <div className="border-b border-zinc-200 px-3 py-2 dark:border-zinc-800">
        <p className="text-xs font-semibold text-zinc-500">Developer details</p>
        <Button type="button" variant="outline" onClick={() => void copyDetails()}>
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className="max-h-48 overflow-auto p-3 text-xs font-mono">{details}</pre>
    </div>
  );
}
