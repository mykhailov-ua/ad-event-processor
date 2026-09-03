import * as React from 'react';

import { Slot } from '@/lib/as_child';
import { adminChrome } from '@/lib/admin_chrome';
import { useControllableState } from '@/lib/controllable_state';
import { anchorBelowTrigger } from '@/lib/floating_position';
import { OverlayRoot } from '@/lib/overlay_root';
import { useOverlayDismiss } from '@/lib/use_overlay_dismiss';
import { cn } from '@/lib/utils';

type MenuContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  triggerRef: React.RefObject<HTMLElement | null>;
};

const MenuContext = React.createContext<MenuContextValue | null>(null);

function useMenuContext() {
  const ctx = React.useContext(MenuContext);
  if (!ctx) {
    throw new Error('DropdownMenu components must be used within <DropdownMenu>');
  }
  return ctx;
}

function DropdownMenu({
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

  return (
    <MenuContext.Provider
      value={{ open: Boolean(isOpen), setOpen: setIsOpen, triggerRef }}
    >
      {children}
    </MenuContext.Provider>
  );
}

const DropdownMenuTrigger = React.forwardRef<
  HTMLElement,
  React.HTMLAttributes<HTMLElement> & { asChild?: boolean }
>(({ asChild = false, className, onClick, children, ...props }, ref) => {
  const { open, setOpen, triggerRef } = useMenuContext();

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
        aria-haspopup="menu"
        className={className}
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
      aria-haspopup="menu"
      className={className}
      onClick={handleClick}
      {...(props as React.ButtonHTMLAttributes<HTMLButtonElement>)}
    >
      {children}
    </button>
  );
});
DropdownMenuTrigger.displayName = 'DropdownMenuTrigger';

const DropdownMenuContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { sideOffset?: number; align?: 'start' | 'end' | 'center' }
>(({ className, sideOffset = 4, align = 'start', style, children, ...props }, ref) => {
  const { open, setOpen, triggerRef } = useMenuContext();
  const contentRef = React.useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = React.useState<React.CSSProperties>({});

  useOverlayDismiss(open, () => setOpen(false), contentRef, [triggerRef]);

  React.useLayoutEffect(() => {
    if (!open || !triggerRef.current) {
      return;
    }
    const rect = triggerRef.current.getBoundingClientRect();
    const base = anchorBelowTrigger(rect, { gap: sideOffset, minWidth: rect.width });
    let left = base.left as number;
    if (align === 'end' && typeof base.minWidth === 'number') {
      left = rect.right - base.minWidth;
    }
    setPosition({ ...base, left });
  }, [align, open, sideOffset, triggerRef]);

  if (!open) {
    return null;
  }

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
        role="menu"
        className={cn(adminChrome.floating, 'min-w-[8rem]', className)}
        style={{ ...position, ...style }}
        {...props}
      >
        {children}
      </div>
    </OverlayRoot>
  );
});
DropdownMenuContent.displayName = 'DropdownMenuContent';

const DropdownMenuItem = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    inset?: boolean;
    onSelect?: (event: Event) => void;
  }
>(({ className, inset, disabled, onClick, onSelect, ...props }, ref) => {
  const { setOpen } = useMenuContext();

  return (
    <button
      ref={ref}
      type="button"
      role="menuitem"
      disabled={disabled}
      className={cn(adminChrome.menuItem, inset && 'pl-6', className)}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented && !disabled) {
          onSelect?.(event.nativeEvent);
          setOpen(false);
        }
      }}
      {...props}
    />
  );
});
DropdownMenuItem.displayName = 'DropdownMenuItem';

const DropdownMenuLabel = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { inset?: boolean }
>(({ className, inset, ...props }, ref) => (
  <div
    ref={ref}
    className={cn('px-2 py-1 text-xs font-semibold text-zinc-500 dark:text-zinc-400', inset && 'pl-6', className)}
    {...props}
  />
));
DropdownMenuLabel.displayName = 'DropdownMenuLabel';

const DropdownMenuSeparator = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('my-1 h-px bg-zinc-200 dark:bg-zinc-800', className)} {...props} />
  ),
);
DropdownMenuSeparator.displayName = 'DropdownMenuSeparator';

const DropdownMenuShortcut = ({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) => (
  <span className={cn('ml-auto text-xs text-zinc-500 dark:text-zinc-400', className)} {...props} />
);
DropdownMenuShortcut.displayName = 'DropdownMenuShortcut';

function DropdownMenuGroup({ children }: { children?: React.ReactNode }) {
  return <div role="group">{children}</div>;
}

function DropdownMenuPortal({ children }: { children?: React.ReactNode }) {
  return <>{children}</>;
}

function DropdownMenuSub({ children }: { children?: React.ReactNode }) {
  return <>{children}</>;
}

function DropdownMenuSubTrigger({
  className,
  inset,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { inset?: boolean }) {
  return (
    <div className={cn(adminChrome.menuItem, inset && 'pl-6', className)} {...props}>
      {children}
      <span aria-hidden className="ml-auto">
        &gt;
      </span>
    </div>
  );
}

function DropdownMenuSubContent({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn(adminChrome.floating, className)} {...props}>
      {children}
    </div>
  );
}

const DropdownMenuCheckboxItem = DropdownMenuItem;
const DropdownMenuRadioItem = DropdownMenuItem;

function DropdownMenuRadioGroup({ children }: { children?: React.ReactNode }) {
  return <div role="radiogroup">{children}</div>;
}

function DropdownMenuSubTriggerChevron() {
  return <span aria-hidden className="ml-auto">&gt;</span>;
}

export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuGroup,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuRadioGroup,
  DropdownMenuSubTriggerChevron,
};
