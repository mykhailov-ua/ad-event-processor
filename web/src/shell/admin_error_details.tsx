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
    <div className="admin-error-details">
      <div className="admin-error-details__header">
        <p className="admin-error-details__title">Developer details</p>
        <Button type="button" variant="outline" onClick={() => void copyDetails()}>
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className="admin-error-details__body">{details}</pre>
    </div>
  );
}
