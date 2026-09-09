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
>(
  (
    { asChild = false, onMouseEnter, onMouseLeave, onFocus, onBlur, children, className, ...props },
    ref
  ) => {
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

        ...handlers,
        ...props,
      });
    }

    return (
      <span className={cn('inline-flex', className)} ref={mergedRef} {...handlers} {...props}>
        {children}
      </span>
    );
  }
);
TooltipTrigger.displayName = 'TooltipTrigger';

const TOOLTIP_FADE_MS = 150;

const TooltipContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    side?: 'top' | 'bottom';
    align?: 'center' | 'start' | 'end';
    sideOffset?: number;
  }
>(
  (
    { side = 'top', align = 'center', sideOffset = 4, className, style, children, ...props },
    ref
  ) => {
    const { open, triggerRef } = useTooltipContext();
    const [coords, setCoords] = React.useState<React.CSSProperties>({});
    const [mounted, setMounted] = React.useState(false);
    const [visible, setVisible] = React.useState(false);

    React.useEffect(() => {
      if (open) {
        setMounted(true);
        const frame = window.requestAnimationFrame(() => {
          setVisible(true);
        });
        return () => {
          window.cancelAnimationFrame(frame);
        };
      }
      setVisible(false);
      const timer = window.setTimeout(() => {
        setMounted(false);
      }, TOOLTIP_FADE_MS);
      return () => {
        window.clearTimeout(timer);
      };
    }, [open]);

    React.useLayoutEffect(() => {
      if (!mounted || !triggerRef.current) {
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
    }, [align, mounted, side, sideOffset, triggerRef]);

    if (!mounted) {
      return null;
    }

    return (
      <OverlayRoot>
        <div
          ref={ref}
          role="tooltip"
          className={cn(
            adminChrome.floating,
            'pointer-events-none z-[10002] whitespace-nowrap px-2 py-1 text-xs transition-opacity duration-150 ease-out',
            visible ? 'opacity-100' : 'opacity-0',
            className
          )}
          style={{ ...coords, ...style }}
          {...props}
        >
          {children}
        </div>
      </OverlayRoot>
    );
  }
);
TooltipContent.displayName = 'TooltipContent';

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
