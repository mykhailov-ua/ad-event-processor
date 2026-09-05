import { Check, Copy } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';

import { buttonVariantClass } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

export type CopyButtonProps = {
  className?: string;
  label?: string;
  value: string;
  /** Swap to checkmark, then hide feedback without layout shift. */
  flashOnCopy?: boolean;
  showToast?: boolean;
};

export function CopyButton({
  className,
  label,
  value,
  flashOnCopy = false,
  showToast = true,
}: CopyButtonProps) {
  const [showCheck, setShowCheck] = useState(false);
  const timersRef = useRef<number[]>([]);
  const trimmed = value.trim();

  useEffect(() => {
    return () => {
      for (const timerId of timersRef.current) {
        window.clearTimeout(timerId);
      }
    };
  }, []);

  if (!trimmed) {
    return null;
  }

  const schedule = (fn: () => void, delayMs: number) => {
    const timerId = window.setTimeout(fn, delayMs);
    timersRef.current.push(timerId);
  };

  const clearTimers = () => {
    for (const timerId of timersRef.current) {
      window.clearTimeout(timerId);
    }
    timersRef.current = [];
  };

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(trimmed);
      clearTimers();
      if (showToast) {
        toast.success(label ? `${label} copied` : 'Copied to clipboard');
      }

      if (flashOnCopy) {
        setShowCheck(true);
        schedule(() => setShowCheck(false), 700);
        return;
      }

      setShowCheck(true);
      schedule(() => setShowCheck(false), 1500);
    } catch {
      toast.error('Could not copy to clipboard');
    }
  };

  return (
    <button
      aria-label={label ? `Copy ${label}` : 'Copy to clipboard'}
      className={cn(
        'inline-flex size-6 shrink-0 items-center justify-center rounded-[5px] border border-transparent p-0',
        'text-muted-foreground transition-none active:scale-100',
        'hover:bg-accent hover:text-foreground',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        buttonVariantClass.ghost,
        className,
      )}
      type="button"
      onClick={() => {
        void onCopy();
      }}
    >
      <span aria-hidden className="relative inline-flex size-4 items-center justify-center">
        <Copy className={cn('h-4 w-4', showCheck && 'invisible')} />
        {showCheck ? <Check className="absolute inset-0 m-auto h-4 w-4" /> : null}
      </span>
    </button>
  );
}
