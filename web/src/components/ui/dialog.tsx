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
  ({ onClick, ...props }, ref) => {
    const { setOpen } = useDialogContext();
    return (
      <div
        ref={ref}
        
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
    panelClassName?: string;
    showCloseButton?: boolean;
  }
>(
  (
    { children,
      onEscapeKeyDown,
      onInteractOutside,
      panelClassName,
      showCloseButton = true,
      ...props
    },
    ref
  ) => {
    const { open, setOpen } = useDialogContext();
    const flush = false;
    const scrollBody = React.Children.toArray(children).some(
      (child) =>
        React.isValidElement(child) &&
        (child.type as { displayName?: string }).displayName === 'DialogBody'
    );
    const useCompactShell = !scrollBody && !flush;

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
              
            >
              {useCompactShell ? <div >{children}</div> : children}
              {showCloseButton ? (
                <button
                  type="button"
                  
                  aria-label="Close"
                  onClick={() => setOpen(false)}
                >
                  <X  />
                </button>
              ) : null}
            </div>
          </DialogLayoutContext.Provider>
        </div>
      </DialogPortal>
    );
  }
);
DialogContent.displayName = 'DialogContent';

const DialogHeader = ({ ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useDialogLayout();

  return (
    <div
      
      {...props}
    />
  );
};
DialogHeader.displayName = 'DialogHeader';

const DialogBody = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ onWheel, ...props }, ref) => (
    <div
      ref={ref}
      
      onWheel={(event) => {
        onWheel?.(event);
        event.stopPropagation();
      }}
      {...props}
    />
  )
);
DialogBody.displayName = 'DialogBody';

const DialogFooter = ({ ...props }: React.HTMLAttributes<HTMLDivElement>) => {
  const { scrollBody } = useDialogLayout();

  return (
    <div
      
      {...props}
    />
  );
};
DialogFooter.displayName = 'DialogFooter';

const DialogTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  ({ ...props }, ref) => (
    <h2 ref={ref}  {...props} />
  )
);
DialogTitle.displayName = 'DialogTitle';

const DialogDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ ...props }, ref) => (
  <p ref={ref}  {...props} />
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
