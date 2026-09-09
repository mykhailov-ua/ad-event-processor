import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { shouldShowAdminErrorDetails } from '@/lib/admin_error';
import { copyTextToClipboard } from '@/lib/copy_text_to_clipboard';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';

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
      await copyTextToClipboard(details);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className={cn(uiSurfaces.panel, 'gap-0 p-0')} >
      <div className={cn(shellChrome.sectionHeaderBandClass, 'p-2')} >
        <p className="m-0 text-xs font-semibold text-muted-foreground" >Developer details</p>
        <Button type="button" variant="outline" onClick={() => void copyDetails()}>
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className="max-h-48 overflow-auto p-3 text-xs font-mono" >{details}</pre>
    </div>
  );
}
