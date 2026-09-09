import { Check, Copy } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';

import { copyTextToClipboard } from '@/lib/copy_text_to_clipboard';
import { directoryTableRowCopyButtonClass } from '@/shell/directory_table_row_actions';
import { cn } from '@/lib/utils';

export type CopyButtonProps = {
  label?: string;
  value: string;
  /** Swap to checkmark, then hide feedback without layout shift. */
  flashOnCopy?: boolean;
  showToast?: boolean;
};

export function CopyButton({ label,
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
      await copyTextToClipboard(trimmed);
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
     
      type="button"
      onClick={() => {
        void onCopy();
      }}
    >
      <span aria-hidden >
        <Copy  />
        {showCheck ? <Check  /> : null}
      </span>
    </button>
  );
}
