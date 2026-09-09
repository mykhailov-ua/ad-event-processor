import * as React from 'react';

import { Slot } from '@/lib/as_child';
import { adminChrome } from '@/lib/admin_chrome';
import { useControllableState } from '@/lib/controllable_state';
import { anchorAboveTrigger, anchorBelowTrigger } from '@/lib/floating_position';
import { subscribeFloatingPosition } from '@/lib/floating_overlay_position';
import { mergeOverlayPosition } from '@/lib/overlay_position_state';
import { OverlayRoot } from '@/lib/overlay_root';
import { useOverlayDismiss } from '@/lib/use_overlay_dismiss';
import { cn } from '@/lib/utils';

type PopoverContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  triggerRef: React.RefObject<HTMLElement | null>;
};

const PopoverContext = React.createContext<PopoverContextValue | null>(null);

function usePopoverContext() {
  const ctx = React.useContext(PopoverContext);
  if (!ctx) {
    throw new Error('Popover components must be used within <Popover>');
  }
  return ctx;
}

function Popover({
  open,
  defaultOpen = false,
  onOpenChange,
  children,
}: {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useControllableState({
    value: open,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const triggerRef = React.useRef<HTMLElement | null>(null);

  const contextValue = React.useMemo(
    () => ({ open: Boolean(isOpen), setOpen: setIsOpen, triggerRef }),
    [isOpen, setIsOpen]
  );

  return (
    <PopoverContext.Provider value={contextValue}>
      {children}
    </PopoverContext.Provider>
  );
}

const PopoverTrigger = React.forwardRef<
  HTMLElement,
  React.HTMLAttributes<HTMLElement> & { asChild?: boolean }
>(({ asChild = false, onClick, children, ...props }, ref) => {
  const { open, setOpen, triggerRef } = usePopoverContext();

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    onClick?.(event);
    if (!event.defaultPrevented) {
      setOpen(!open);
    }
  };

  const mergedRef = (node: HTMLElement | null) => {
    triggerRef.current = node;
    if (typeof ref === 'function') {
      ref(node);
    } else if (ref) {
      ref.current = node;
    }
  };

  if (asChild && React.isValidElement(children)) {
    return (
      <Slot
        ref={mergedRef}
        aria-expanded={open}
        
        onClick={handleClick}
        {...props}
      >
        {children}
      </Slot>
    );
  }

  return (
    <button
      ref={mergedRef as React.Ref<HTMLButtonElement>}
      type="button"
      aria-expanded={open}
      
      onClick={handleClick}
      {...(props as React.ButtonHTMLAttributes<HTMLButtonElement>)}
    >
      {children}
    </button>
  );
});
PopoverTrigger.displayName = 'PopoverTrigger';

const PopoverAnchor = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ ...props }, ref) => (
    <div ref={ref}  {...props} />
  )
);
PopoverAnchor.displayName = 'PopoverAnchor';

const PopoverContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    align?: 'start' | 'center' | 'end';
    side?: 'top' | 'bottom' | 'left' | 'right';
    sideOffset?: number;
    collisionPadding?: number;
    panelScroll?: 'panel' | 'inner' | 'none';
    panelClassName?: string;
    matchTriggerMinWidth?: boolean;
    onOpenAutoFocus?: (event: Event) => void;
    sticky?: string;
  }
>(
  (
    { children,
      align = 'center',
      side = 'bottom',
      sideOffset = 8,
      panelScroll,
      panelClassName,
      matchTriggerMinWidth = true,
      onOpenAutoFocus,
      ...props
    },
    ref
  ) => {
    const { open, setOpen, triggerRef } = usePopoverContext();
    const contentRef = React.useRef<HTMLDivElement | null>(null);
    const [position, setPosition] = React.useState<React.CSSProperties>({});
    const flush = false;
    const scrollMode = panelScroll ?? (flush ? 'inner' : 'panel');

    useOverlayDismiss(open, () => setOpen(false), contentRef, [triggerRef]);

    React.useEffect(() => {
      if (open && onOpenAutoFocus) {
        onOpenAutoFocus(new Event('focus'));
      }
    }, [onOpenAutoFocus, open]);

    React.useLayoutEffect(() => {
      if (!open || !triggerRef.current) {
        return;
      }

      const edgePadding = 12;

      const updatePosition = () => {
        const trigger = triggerRef.current;
        const content = contentRef.current;
        if (!trigger || !content) {
          return;
        }
        const rect = trigger.getBoundingClientRect();
        const panel = content.firstElementChild as HTMLElement | null;
        const contentWidth = panel?.offsetWidth ?? content.offsetWidth;
        const contentHeight = panel?.offsetHeight ?? content.offsetHeight;
        const base =
          side === 'top'
            ? anchorAboveTrigger(rect, contentHeight, { gap: sideOffset, minWidth: rect.width })
            : anchorBelowTrigger(rect, { gap: sideOffset, minWidth: rect.width });
        let left = rect.left;
        if (align === 'center') {
          left = rect.left + rect.width / 2 - contentWidth / 2;
        } else if (align === 'end') {
          left = rect.right - contentWidth;
        }
        left = Math.max(
          edgePadding,
          Math.min(left, window.innerWidth - contentWidth - edgePadding)
        );
        setPosition((prev) => mergeOverlayPosition(prev, { ...base, left }));
      };

      updatePosition();
      const raf = window.requestAnimationFrame(updatePosition);
      const unsubscribeScroll = subscribeFloatingPosition(triggerRef.current, updatePosition);
      window.addEventListener('resize', updatePosition);

      const panel = contentRef.current?.firstElementChild;
      const resizeObserver =
        typeof ResizeObserver !== 'undefined' && panel
          ? new ResizeObserver(() => updatePosition())
          : undefined;
      if (resizeObserver && panel) {
        resizeObserver.observe(panel);
      }

      return () => {
        window.cancelAnimationFrame(raf);
        unsubscribeScroll();
        window.removeEventListener('resize', updatePosition);
        resizeObserver?.disconnect();
      };
    }, [align, open, side, sideOffset, triggerRef]);

    if (!open) {
      return null;
    }

    const {
      className,
      style,
      ...contentProps
    } = props;

    const panelClass = cn(
      adminChrome.floating,
      matchTriggerMinWidth && 'min-w-[var(--popover-anchor-width,12rem)]',
      scrollMode === 'panel' && 'max-h-[min(28rem,calc(100dvh-2rem))] overflow-y-auto',
      panelClassName
    );

    const anchorWidth = `${triggerRef.current?.offsetWidth ?? 0}px`;
    const anchorWidthStyle = {
      '--popover-anchor-width': anchorWidth,
    } as React.CSSProperties;

    return (
      <OverlayRoot>
        <div
          ref={(node) => {
            contentRef.current = node;
            if (typeof ref === 'function') {
              ref(node);
            } else if (ref) {
              ref.current = node;
            }
          }}
          className={cn('fixed z-50', className)}
          style={{ ...position, ...style, ...anchorWidthStyle }}
          {...contentProps}
        >
          <div className={panelClass} style={anchorWidthStyle}>
            <div className={cn(scrollMode === 'inner' && 'max-h-[min(28rem,calc(100dvh-2rem))] overflow-y-auto')}>
              {children}
            </div>
          </div>
        </div>
      </OverlayRoot>
    );
  }
);
PopoverContent.displayName = 'PopoverContent';

export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor };
