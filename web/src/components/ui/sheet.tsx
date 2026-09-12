import * as React from 'react';
import { X } from 'lucide-react';

import { Slot } from '@/lib/as_child';
import { adminChrome } from '@/lib/admin_chrome';
import { useControllableState } from '@/lib/controllable_state';
import { OverlayRoot } from '@/lib/overlay_root';
import { cn } from '@/lib/utils';

export type SheetProps = {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
};

type SheetContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
};

const SheetContext = React.createContext<SheetContextValue | null>(null);

type SheetLayoutContextValue = {
  scrollBody: boolean;
};

const SheetLayoutContext = React.createContext<SheetLayoutContextValue>({ scrollBody: false });

function useSheetLayout() {
  return React.useContext(SheetLayoutContext);
}

function useSheetContext() {
  const ctx = React.useContext(SheetContext);
  if (!ctx) {
    throw new Error('Sheet compound components must be used within <Sheet>');
  }
  return ctx;
}

function Sheet({ open, defaultOpen = false, onOpenChange, children }: SheetProps) {
  const [isOpen, setIsOpen] = useControllableState({
    value: open,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });

  return (
    <SheetContext.Provider value={{ open: Boolean(isOpen), setOpen: setIsOpen }}>
      {children}
    </SheetContext.Provider>
  );
}

function SheetTrigger({
  asChild = false,
  onClick,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { asChild?: boolean }) {
  const { setOpen } = useSheetContext();

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    onClick?.(event);
    if (!event.defaultPrevented) {
      setOpen(true);
    }
  };

  if (asChild) {
    return <Slot onClick={handleClick} {...props} />;
  }

  return <button type="button" onClick={handleClick} {...props} />;
}

function SheetClose({
  asChild = false,
  onClick,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { asChild?: boolean }) {
  const { setOpen } = useSheetContext();

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    onClick?.(event);
    if (!event.defaultPrevented) {
      setOpen(false);
    }
  };

  if (asChild) {
    return <Slot onClick={handleClick} {...props} />;
  }

  return <button type="button" onClick={handleClick} {...props} />;
}

const SheetPortal = ({ children }: { children: React.ReactNode }) => (
  <OverlayRoot>{children}</OverlayRoot>
);

const SheetOverlay = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, onClick, ...props }, ref) => {
    const { setOpen } = useSheetContext();
    return (
      <div
        ref={ref}
        className={cn(adminChrome.overlayBackdrop, className)}
        onClick={(event) => {
          onClick?.(event);
          setOpen(false);
        }}
        {...props}
      />
    );
  }
);
SheetOverlay.displayName = 'SheetOverlay';

type SheetSide = 'top' | 'bottom' | 'left' | 'right';

const sideClass: Record<SheetSide, string> = {
  top: 'inset-x-0 top-0 border-b',
  bottom: 'inset-x-0 bottom-0 border-t',
  left: 'inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm',
  right: 'inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm',
};

const SheetContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { side?: SheetSide }
>(({ side = 'right', className, children, ...props }, ref) => {
  const { open, setOpen } = useSheetContext();
  const scrollBody = React.Children.toArray(children).some(
    (child) =>
      React.isValidElement(child) &&
      (child.type as { displayName?: string }).displayName === 'SheetBody'
  );

  if (!open) {
    return null;
  }

  return (
    <SheetPortal>
      <SheetOverlay />
      <SheetLayoutContext.Provider value={{ scrollBody }}>
        <div
          ref={ref}
          className={cn(
            'fixed z-50 flex min-h-0 flex-col gap-0 border-border bg-card text-card-foreground shadow-lg',
            scrollBody ? 'overflow-hidden' : 'ui-scrollbar gap-4 overflow-y-auto p-6',
            sideClass[side],
            className
          )}
          role="dialog"
          aria-modal="true"
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              event.preventDefault();
              setOpen(false);
            }
          }}
          {...props}
        >
          {children}
          <button
            type="button"
            className="absolute right-4 top-4 z-10 rounded-sm p-1 text-muted-foreground hover:text-foreground"
            aria-label="Close"
            onClick={() => setOpen(false)}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </SheetLayoutContext.Provider>
    </SheetPortal>
  );
});
SheetContent.displayName = 'SheetContent';

const SheetHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useSheetLayout();

  return (
    <div
      className={cn(
        'flex w-full shrink-0 flex-col items-start gap-2 text-left',
        scrollBody && 'px-6 pt-6',
        className
      )}
      {...props}
    />
  );
};
SheetHeader.displayName = 'SheetHeader';

const SheetBody = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, onWheel, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'ui-scrollbar min-h-0 flex-1 overflow-y-auto overscroll-y-contain px-6 py-4',
        className
      )}
      onWheel={(event) => {
        onWheel?.(event);
        event.stopPropagation();
      }}
      {...props}
    />
  )
);
SheetBody.displayName = 'SheetBody';

const SheetFooter = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useSheetLayout();

  return (
    <div
      className={cn(
        'flex w-full shrink-0 flex-col-reverse items-stretch gap-2 sm:flex-row sm:justify-end',
        scrollBody && 'px-6 pb-6',
        className
      )}
      {...props}
    />
  );
};
SheetFooter.displayName = 'SheetFooter';

const SheetTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h2 ref={ref} className={cn('text-lg font-semibold text-foreground', className)} {...props} />
  )
);
SheetTitle.displayName = 'SheetTitle';

const SheetDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p ref={ref} className={cn('text-sm text-muted-foreground', className)} {...props} />
));
SheetDescription.displayName = 'SheetDescription';

export {
  Sheet,
  SheetPortal,
  SheetOverlay,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetBody,
  SheetFooter,
  SheetTitle,
  SheetDescription,
};
