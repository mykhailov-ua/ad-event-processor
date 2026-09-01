import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type CopyButtonProps = {
  className?: string;
  label?: string;
  value: string;
};

export function CopyButton({ className, label, value }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const trimmed = value.trim();
  const disabled = trimmed.length === 0;

  const onCopy = async () => {
    if (disabled) {
      return;
    }
    try {
      await navigator.clipboard.writeText(trimmed);
      setCopied(true);
      toast.success(label ? `${label} copied` : 'Copied to clipboard');
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error('Could not copy to clipboard');
    }
  };

  return (
    <Button
      aria-label={label ? `Copy ${label}` : 'Copy to clipboard'}
      className={cn('size-8 shrink-0 rounded-full p-0', className)}
      disabled={disabled}
      onClick={() => {
        void onCopy();
      }}
      size="icon"
      type="button"
      variant="ghost"
    >
      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
    </Button>
  );
}
