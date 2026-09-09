import * as React from 'react';
import { Check, ChevronDown } from 'lucide-react';

import { Slot } from '@/lib/as_child';
import { Button } from '@/components/ui/button';
import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { useControllableState } from '@/lib/controllable_state';
import {
  computeFloatingPosition,
  subscribeFloatingPosition,
} from '@/lib/floating_overlay_position';
import { mergeOverlayPosition } from '@/lib/overlay_position_state';
import { OverlayRoot } from '@/lib/overlay_root';
import { useOverlayDismiss } from '@/lib/use_overlay_dismiss';
import { cn } from '@/lib/utils';

type SelectContextValue = {
  open: boolean;
  setOpen: (open: boolean) => void;
  value?: string;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
  labels: Map<string, string>;
  registerLabel: (value: string, label: string) => void;
  triggerRef: React.RefObject<HTMLButtonElement | null>;
};

const SelectContext = React.createContext<SelectContextValue | null>(null);

function useSelectContext() {
  const ctx = React.useContext(SelectContext);
  if (!ctx) {
    throw new Error('Select components must be used within <Select>');
  }
  return ctx;
}

type SelectProps = {
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
  children?: React.ReactNode;
};

function Select({
  value,
  defaultValue,
  onValueChange,
  disabled,
  children,
}: SelectProps) {
  const [open, setOpen] = React.useState(false);
  const [internalValue, setInternalValue] = useControllableState({
    value,
    defaultValue,
    onChange: onValueChange,
  });
  const [labels, setLabels] = React.useState(() => new Map<string, string>());
  const triggerRef = React.useRef<HTMLButtonElement | null>(null);

  const registerLabel = React.useCallback((itemValue: string, label: string) => {
    setLabels((current) => {
      if (current.get(itemValue) === label) {
        return current;
      }
      const next = new Map(current);
      next.set(itemValue, label);
      return next;
    });
  }, []);

  const contextValue = React.useMemo(
    () => ({
      open,
      setOpen,
      value: internalValue,
      onValueChange: setInternalValue,
      disabled,
      labels,
      registerLabel,
      triggerRef,
    }),
    [disabled, internalValue, labels, open, registerLabel, setInternalValue]
  );

  return <SelectContext.Provider value={contextValue}>{children}</SelectContext.Provider>;
}

function SelectGroup({ children }: { children?: React.ReactNode }) {
  return <div role="group">{children}</div>;
}

const SelectValue = ({ placeholder }: { placeholder?: string }) => {
  const { value, labels } = useSelectContext();
  const registered = value ? labels.get(value) : undefined;
  const fallback =
    value && /^__.*__$/.test(value)
      ? undefined
      : value && value.length <= 8
        ? value.toUpperCase()
        : value;
  const label = registered ?? fallback;
  return (
    <span className="truncate text-left">
      {label ?? placeholder ?? 'Select...'}
    </span>
  );
};

const SelectTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & { plain?: boolean }
>(({ children, plain = false, disabled, className, onClick, ...props }, ref) => {
  const ctx = useSelectContext();
  const isDisabled = disabled ?? ctx.disabled;

  const mergedRef = (node: HTMLButtonElement | null) => {
    ctx.triggerRef.current = node;
    if (typeof ref === 'function') {
      ref(node);
    } else if (ref) {
      ref.current = node;
    }
  };

  if (plain) {
    return (
      <Button
        ref={mergedRef}
        type="button"
        variant="ghost"
        disabled={isDisabled}
        aria-expanded={ctx.open}
        aria-haspopup="listbox"
        className={cn('w-full justify-between font-normal', className)}
        onClick={(event) => {
          onClick?.(event);
          if (!event.defaultPrevented && !isDisabled) {
            ctx.setOpen(!ctx.open);
          }
        }}
        {...props}
      >
        {children}
        <ChevronDown className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
      </Button>
    );
  }

  return (
    <button
      ref={mergedRef}
      type="button"
      disabled={isDisabled}
      aria-expanded={ctx.open}
      aria-haspopup="listbox"
      className={cn(
        adminChrome.control,
        'inline-flex w-full items-center justify-between gap-2 text-left font-normal',
        className
      )}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented && !isDisabled) {
          ctx.setOpen(!ctx.open);
        }
      }}
      {...props}
    >
      {children}
      <ChevronDown className="h-4 w-4 shrink-0 opacity-60" aria-hidden />
    </button>
  );
});
SelectTrigger.displayName = 'SelectTrigger';

const SelectContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { plain?: boolean; position?: 'popper' | 'item-aligned' }
>(({ className, children, plain = false, position = 'popper', ...props }, ref) => {
  const { open, setOpen, triggerRef } = useSelectContext();
  const contentRef = React.useRef<HTMLDivElement | null>(null);
  const [coords, setCoords] = React.useState<React.CSSProperties>({
    position: 'fixed',
    visibility: 'hidden',
  });

  useOverlayDismiss(open, () => setOpen(false), contentRef, [triggerRef]);

  React.useLayoutEffect(() => {
    if (!open || !triggerRef.current) {
      return;
    }

    const updatePosition = () => {
      const trigger = triggerRef.current;
      const content = contentRef.current;
      if (!trigger || !content) {
        return;
      }
      const rect = trigger.getBoundingClientRect();
      const width = Math.max(content.offsetWidth, rect.width);
      const next = computeFloatingPosition(rect, width, content.offsetHeight, {
        align: 'start',
        gap: 4,
      });
      setCoords((prev) =>
        mergeOverlayPosition(prev, {
          ...next,
          minWidth: position === 'popper' ? rect.width : undefined,
          visibility: 'visible',
        })
      );
    };

    updatePosition();
    const raf = window.requestAnimationFrame(updatePosition);
    const unsubscribeScroll = subscribeFloatingPosition(triggerRef.current, updatePosition);
    const resizeObserver =
      typeof ResizeObserver !== 'undefined' && contentRef.current
        ? new ResizeObserver(() => updatePosition())
        : undefined;
    if (resizeObserver && contentRef.current) {
      resizeObserver.observe(contentRef.current);
    }

    return () => {
      window.cancelAnimationFrame(raf);
      unsubscribeScroll();
      resizeObserver?.disconnect();
    };
  }, [open, position, triggerRef]);

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
        role="listbox"
        className={cn(adminChrome.floating, 'max-h-96 w-max overflow-hidden p-0', className)}
        style={coords}
        {...props}
      >
        <div
          className={cn(adminChrome.menuList, 'scrollbar-admin max-h-60 overflow-y-auto p-1')}
        >
          {children}
        </div>
      </div>
    </OverlayRoot>
  );
});
SelectContent.displayName = 'SelectContent';

const SelectLabel = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(adminKit.labelCaps, 'px-2 py-1.5', className)}
      {...props}
    />
  )
);
SelectLabel.displayName = 'SelectLabel';

const SelectItem = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & { plain?: boolean; value: string }
>(({ className, children, plain = false, value, disabled, onClick, ...props }, ref) => {
  const { registerLabel, value: selectedValue, onValueChange, setOpen } = useSelectContext();
  const label = typeof children === 'string' ? children : value;

  React.useEffect(() => {
    registerLabel(value, label);
  }, [label, registerLabel, value]);

  const selected = selectedValue === value;

  return (
    <button
      ref={ref}
      type="button"
      role="option"
      aria-selected={selected}
      disabled={disabled}
      className={cn(
        adminChrome.menuItem,
        'relative pr-8',
        selected && adminChrome.menuItemSelected,
        className
      )}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented && !disabled) {
          onValueChange?.(value);
          setOpen(false);
        }
      }}
      {...props}
    >
      {children}
      {!plain && selected ? (
        <span className="absolute right-2 flex h-3.5 w-3.5 items-center justify-center text-foreground">
          <Check className="h-4 w-4" strokeWidth={2.5} />
        </span>
      ) : null}
    </button>
  );
});
SelectItem.displayName = 'SelectItem';

const SelectSeparator = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn('my-1 h-px bg-border', className)} {...props} />
  )
);
SelectSeparator.displayName = 'SelectSeparator';

function SelectScrollUpButton() {
  return null;
}

function SelectScrollDownButton() {
  return null;
}

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
  SelectScrollUpButton,
  SelectScrollDownButton,
};
