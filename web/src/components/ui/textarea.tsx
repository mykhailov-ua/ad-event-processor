import * as React from 'react';
import { useCallback, useLayoutEffect, useRef } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

export type TextareaProps = React.ComponentProps<'textarea'> & {
  maxLength?: number;
  showCount?: boolean;
};

function mergeRefs<T>(...refs: Array<React.Ref<T> | undefined>) {
  return (node: T | null) => {
    for (const ref of refs) {
      if (!ref) {
        continue;
      }
      if (typeof ref === 'function') {
        ref(node);
      } else {
        (ref as React.MutableRefObject<T | null>).current = node;
      }
    }
  };
}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, maxLength, showCount, value, onChange, rows = 3, ...props }, ref) => {
    const innerRef = useRef<HTMLTextAreaElement | null>(null);
    const showCounter = showCount ?? maxLength != null;
    const length = typeof value === 'string' ? value.length : 0;

    const syncHeight = useCallback(() => {
      const element = innerRef.current;
      if (!element) {
        return;
      }
      element.style.height = 'auto';
      element.style.height = `${element.scrollHeight}px`;
    }, []);

    useLayoutEffect(() => {
      syncHeight();
    }, [syncHeight, value]);

    return (
      <div className="relative">
        <div className={cn(adminChrome.panel, 'overflow-hidden')}>
          <textarea
            ref={mergeRefs(ref, innerRef)}
            rows={rows}
            maxLength={maxLength}
            value={value}
            onChange={(event) => {
              onChange?.(event);
              syncHeight();
            }}
            className={cn(
              'flex min-h-[5rem] w-full resize-none overflow-hidden border-0 bg-transparent px-3 py-2 font-mono text-sm text-foreground transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
              showCounter && maxLength != null && 'pb-7',
              className,
            )}
            {...props}
          />
        </div>
        {showCounter && maxLength != null ? (
          <span
            className="pointer-events-none absolute bottom-2 right-3 text-xs tabular-nums text-muted-foreground"
            aria-hidden="true"
          >
            {length}/{maxLength.toLocaleString()}
          </span>
        ) : null}
      </div>
    );
  },
);
Textarea.displayName = 'Textarea';

export { Textarea };
