import * as React from 'react';
import { X } from 'lucide-react';

import { Slot } from '@/lib/as_child';
import { adminChrome } from '@/lib/admin_chrome';
import { useControllableState } from '@/lib/controllable_state';
import { OverlayRoot } from '@/lib/overlay_root';
import { cn } from '@/lib/utils';

export type DialogProps = {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
};

type DialogContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
};

const DialogContext = React.createContext<DialogContextValue | null>(null);

type DialogLayoutContextValue = {
  scrollBody: boolean;
};

const DialogLayoutContext = React.createContext<DialogLayoutContextValue>({ scrollBody: false });

function useDialogLayout() {
  return React.useContext(DialogLayoutContext);
}

function useDialogContext() {
  const ctx = React.useContext(DialogContext);
  if (!ctx) {
    throw new Error('Dialog compound components must be used within <Dialog>');
  }
  return ctx;
}

function Dialog({ open, defaultOpen = false, onOpenChange, children }: DialogProps) {
  const [isOpen, setIsOpen] = useControllableState({
    value: open,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });

  return (
    <DialogContext.Provider value={{ open: Boolean(isOpen), setOpen: setIsOpen }}>
      {children}
    </DialogContext.Provider>
  );
}

type DialogTriggerProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  asChild?: boolean;
};

function DialogTrigger({ asChild = false, onClick, ...props }: DialogTriggerProps) {
  const { setOpen } = useDialogContext();

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

function DialogClose({
  asChild = false,
  onClick,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { asChild?: boolean }) {
  const { setOpen } = useDialogContext();

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

const DialogPortal = ({ children }: { children: React.ReactNode }) => (
  <OverlayRoot>{children}</OverlayRoot>
);

const DialogOverlay = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, onClick, ...props }, ref) => {
    const { setOpen } = useDialogContext();
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
DialogOverlay.displayName = 'DialogOverlay';

const DialogContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & {
    onEscapeKeyDown?: (event: KeyboardEvent) => void;
    onInteractOutside?: (event: Event) => void;
  }
>(({ className, children, onEscapeKeyDown, onInteractOutside, ...props }, ref) => {
  const { open, setOpen } = useDialogContext();
  const flush = /\bp-0\b/.test(className ?? '');
  const scrollBody = React.Children.toArray(children).some(
    (child) =>
      React.isValidElement(child) &&
      (child.type as { displayName?: string }).displayName === 'DialogBody'
  );

  if (!open) {
    return null;
  }

  return (
    <DialogPortal>
      <DialogOverlay
        onClick={(event) => {
          if (onInteractOutside) {
            onInteractOutside(event.nativeEvent);
            if (event.defaultPrevented) {
              return;
            }
          }
          setOpen(false);
        }}
      />
      <div
        ref={ref}
        className={cn(
          'fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 border-0 bg-transparent p-0',
          className
        )}
        role="dialog"
        aria-modal="true"
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            if (onEscapeKeyDown) {
              onEscapeKeyDown(event.nativeEvent);
              if (event.defaultPrevented) {
                return;
              }
            }
            event.preventDefault();
            setOpen(false);
          }
        }}
        {...props}
      >
        <DialogLayoutContext.Provider value={{ scrollBody }}>
          <div
            className={cn(
              adminChrome.panel,
              'relative w-full shadow-lg',
              scrollBody
                ? 'flex max-h-[min(90vh,48rem)] flex-col gap-0 overflow-hidden'
                : flush
                  ? 'overflow-hidden'
                  : 'ui-scrollbar max-h-[min(90vh,48rem)] gap-4 overflow-y-auto p-6'
            )}
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
        </DialogLayoutContext.Provider>
      </div>
    </DialogPortal>
  );
});
DialogContent.displayName = 'DialogContent';

const DialogHeader = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useDialogLayout();

  return (
    <div
      className={cn(
        'flex shrink-0 flex-col gap-1.5 text-center sm:text-left',
        scrollBody && 'px-6 pt-6',
        className
      )}
      {...props}
    />
  );
};
DialogHeader.displayName = 'DialogHeader';

const DialogBody = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, onWheel, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'ui-scrollbar min-h-0 flex-1 overflow-y-auto overscroll-y-contain px-6 pt-4 pb-6',
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
DialogBody.displayName = 'DialogBody';

const DialogFooter = ({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useDialogLayout();

  return (
    <div
      className={cn(
        'flex shrink-0 flex-col-reverse gap-2 sm:flex-row sm:justify-end',
        scrollBody && 'px-6 pb-6',
        className
      )}
      {...props}
    />
  );
};
DialogFooter.displayName = 'DialogFooter';

const DialogTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => (
    <h2 ref={ref} className={cn(adminChrome.pageTitle, 'text-lg', className)} {...props} />
  )
);
DialogTitle.displayName = 'DialogTitle';

const DialogDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
  <p ref={ref} className={cn('text-sm text-muted-foreground', className)} {...props} />
));
DialogDescription.displayName = 'DialogDescription';

export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogTrigger,
  DialogClose,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogFooter,
  DialogTitle,
  DialogDescription,
};
