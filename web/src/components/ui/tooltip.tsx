import * as React from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { mergeRefs } from '@/lib/as_child';
import { OverlayRoot } from '@/lib/overlay_root';
import { cn } from '@/lib/utils';
import { computeTooltipCoords } from '@/components/ui/tooltip_position';

function TooltipProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}

type TooltipContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  triggerRef: React.RefObject<HTMLElement | null>;
};

const TooltipContext = React.createContext<TooltipContextValue | null>(null);

function useTooltipContext() {
  const ctx = React.useContext(TooltipContext);
  if (!ctx) {
    throw new Error('Tooltip components must be used within <Tooltip>');
  }
  return ctx;
}

function Tooltip({ children }: { children?: React.ReactNode }) {
  const [open, setOpen] = React.useState(false);
  const triggerRef = React.useRef<HTMLElement | null>(null);

  return (
    <TooltipContext.Provider value={{ open, setOpen, triggerRef }}>
      {children}
    </TooltipContext.Provider>
  );
}

const TooltipTrigger = React.forwardRef<
  HTMLElement,
  React.HTMLAttributes<HTMLElement> & { asChild?: boolean }
>(({ asChild = false, className, onMouseEnter, onMouseLeave, onFocus, onBlur, children, ...props }, ref) => {
  const { setOpen, triggerRef } = useTooltipContext();

  const mergedRef = (node: HTMLElement | null) => {
    triggerRef.current = node;
    if (typeof ref === 'function') {
      ref(node);
    } else if (ref) {
      ref.current = node;
    }
  };

  const handlers = {
    onMouseEnter: (event: React.MouseEvent<HTMLElement>) => {
      onMouseEnter?.(event);
      setOpen(true);
    },
    onMouseLeave: (event: React.MouseEvent<HTMLElement>) => {
      onMouseLeave?.(event);
      setOpen(false);
    },
    onFocus: (event: React.FocusEvent<HTMLElement>) => {
      onFocus?.(event);
      setOpen(true);
    },
    onBlur: (event: React.FocusEvent<HTMLElement>) => {
      onBlur?.(event);
      setOpen(false);
    },
  };

  if (asChild && React.isValidElement(children)) {
    const child = children as React.ReactElement<Record<string, unknown>>;
    const childRef = (child as { ref?: React.Ref<HTMLElement> }).ref;
    return React.cloneElement(child, {
      ref: mergeRefs(mergedRef, childRef),
      className: cn(className, (children.props as { className?: string }).className),
      ...handlers,
      ...props,
    });
  }

  return (
    <span
      ref={mergedRef}
      className={cn('inline-flex', className)}
      {...handlers}
      {...props}
    >
      {children}
    </span>
  );
});
TooltipTrigger.displayName = 'TooltipTrigger';

const TooltipContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    side?: 'top' | 'bottom';
    align?: 'center' | 'start' | 'end';
    sideOffset?: number;
  }
>(({ className, side = 'top', align = 'center', sideOffset = 4, style, children, ...props }, ref) => {
  const { open, triggerRef } = useTooltipContext();
  const [coords, setCoords] = React.useState<React.CSSProperties>({});

  React.useLayoutEffect(() => {
    if (!open || !triggerRef.current) {
      return;
    }

    const updateCoords = () => {
      const node = triggerRef.current;
      if (!node) {
        return;
      }
      setCoords(computeTooltipCoords(node.getBoundingClientRect(), side, align, sideOffset));
    };

    updateCoords();
    window.addEventListener('scroll', updateCoords, true);
    window.addEventListener('resize', updateCoords);
    return () => {
      window.removeEventListener('scroll', updateCoords, true);
      window.removeEventListener('resize', updateCoords);
    };
  }, [align, open, side, sideOffset, triggerRef]);

  if (!open) {
    return null;
  }

  return (
    <OverlayRoot>
      <div
        ref={ref}
        role="tooltip"
        className={cn(
          adminChrome.panel,
          'pointer-events-none px-3 py-1.5 text-xs text-foreground shadow-lg',
          className,
        )}
        style={{ ...coords, ...style }}
        {...props}
      >
        {children}
      </div>
    </OverlayRoot>
  );
});
TooltipContent.displayName = 'TooltipContent';

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
