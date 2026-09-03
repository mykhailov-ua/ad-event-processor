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
  },
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

  if (!open) {
    return null;
  }

  return (
    <SheetPortal>
      <SheetOverlay />
      <div
        ref={ref}
        className={cn(
          'fixed z-50 flex flex-col gap-4 border-zinc-200 bg-white p-6 shadow-lg dark:border-zinc-800 dark:bg-zinc-950',
          sideClass[side],
          className,
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
          className="absolute right-4 top-4 rounded-sm p-1 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          aria-label="Close"
          onClick={() => setOpen(false)}
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </SheetPortal>
  );
});
SheetContent.displayName = 'SheetContent';

const SheetHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div className={cn('flex flex-col gap-2 text-center sm:text-left', className)} {...props} />
);
SheetHeader.displayName = 'SheetHeader';

const SheetFooter = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
  <div className={cn('flex flex-col-reverse gap-2 sm:flex-row sm:justify-end', className)} {...props} />
);
SheetFooter.displayName = 'SheetFooter';

const SheetTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h2 ref={ref} className={cn('text-lg font-semibold text-zinc-900 dark:text-zinc-50', className)} {...props} />
  ),
);
SheetTitle.displayName = 'SheetTitle';

const SheetDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p ref={ref} className={cn('text-sm text-zinc-500 dark:text-zinc-400', className)} {...props} />
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
  SheetFooter,
  SheetTitle,
  SheetDescription,
};
