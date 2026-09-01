import * as React from 'react';
import * as PopoverPrimitive from '@radix-ui/react-popover';

import { cn } from '@/lib/utils';

const Popover = PopoverPrimitive.Root;
const PopoverTrigger = PopoverPrimitive.Trigger;
const PopoverAnchor = PopoverPrimitive.Anchor;

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content> & {
    panelScroll?: 'panel' | 'inner';
  }
>(({ className, children, align = 'center', sideOffset = 8, collisionPadding = 16, panelScroll, ...props }, ref) => {
  const flush = /\bp-0\b/.test(className ?? '');
  const scrollMode = panelScroll ?? (flush ? 'inner' : 'panel');

  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        ref={ref}
        align={align}
        sideOffset={sideOffset}
        collisionPadding={collisionPadding}
        avoidCollisions
        sticky="partial"
        className={cn(
          'z-50 w-auto border-0 bg-transparent p-0 outline-none',
          'data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
          className,
        )}
        {...props}
      >
        <div
          className={cn(
            'ui-shell min-w-[var(--radix-popover-trigger-width)]',
            flush &&
              'w-auto max-w-[min(calc(100vw-2rem),var(--radix-popover-content-available-width))]',
            !flush && 'w-[var(--radix-popover-trigger-width)]',
          )}
        >
          <div
            className={cn(
              'ui-shell-panel',
              scrollMode === 'panel' &&
                'ui-scrollbar max-h-[min(70vh,var(--radix-popover-content-available-height))] overflow-y-auto overflow-x-auto',
              scrollMode === 'inner' &&
                'max-h-[min(70vh,var(--radix-popover-content-available-height))] overflow-hidden',
              !flush && 'p-4',
            )}
          >
            {children}
          </div>
        </div>
      </PopoverPrimitive.Content>
    </PopoverPrimitive.Portal>
  );
});
PopoverContent.displayName = PopoverPrimitive.Content.displayName;

export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor };
