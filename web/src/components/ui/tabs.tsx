import * as React from 'react';

import { useControllableState } from '@/lib/controllable_state';
import { cn } from '@/lib/utils';

function Tabs({
  value,
  defaultValue,
  onValueChange,
  children,
  className,
}: {
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  children?: React.ReactNode;
  className?: string;
}) {
  const [active, setActive] = useControllableState({
    value,
    defaultValue,
    onChange: onValueChange,
  });

  return (
    <TabsContext.Provider value={{ active: active ?? '', setActive }}>
      <div className={className}>{children}</div>
    </TabsContext.Provider>
  );
}

type TabsContextValue = {
  active: string;
  setActive: (value: string) => void;
};

const TabsContext = React.createContext<TabsContextValue | null>(null);

function useTabsContext() {
  const ctx = React.useContext(TabsContext);
  if (!ctx) {
    throw new Error('Tabs components must be used within <Tabs>');
  }
  return ctx;
}

const TabsList = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { variant?: 'segmented' | 'pill' | 'underline' }
>(({ className, variant = 'segmented', ...props }, ref) => (
  <div
    ref={ref}
    role="tablist"
    className={cn(
      variant === 'segmented' &&
        'inline-flex h-9 items-center rounded-[5px] bg-muted p-1 text-muted-foreground',
      variant === 'pill' && 'inline-flex flex-wrap items-center gap-2',
      variant === 'underline' &&
        'inline-flex items-center gap-4 border-b border-border',
      className,
    )}
    {...props}
  />
));
TabsList.displayName = 'TabsList';

const TabsTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & { value: string; variant?: 'segmented' | 'pill' | 'underline' }
>(({ className, value, variant = 'segmented', onClick, ...props }, ref) => {
  const { active, setActive } = useTabsContext();
  const selected = active === value;

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={selected}
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap text-[13px] font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
        variant === 'segmented' && 'rounded-[5px] px-3 py-1',
        variant === 'segmented' &&
          (selected
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'),
        variant === 'pill' &&
          cn(
            'min-h-7 rounded-full border px-3 py-1',
            selected
              ? 'border-primary bg-primary text-primary-foreground'
              : 'border-border bg-background text-foreground hover:bg-accent',
          ),
        variant === 'underline' &&
          cn(
            'border-b-2 px-0 pb-2',
            selected
              ? 'border-primary text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground',
          ),
        className,
      )}
      onClick={(event) => {
        onClick?.(event);
        if (!event.defaultPrevented) {
          setActive(value);
        }
      }}
      {...props}
    />
  );
});
TabsTrigger.displayName = 'TabsTrigger';

const TabsContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { value: string }
>(({ className, value, ...props }, ref) => {
  const { active } = useTabsContext();
  if (active !== value) {
    return null;
  }

  return (
    <div ref={ref} role="tabpanel" className={cn('mt-2', className)} {...props} />
  );
});
TabsContent.displayName = 'TabsContent';

export { Tabs, TabsList, TabsTrigger, TabsContent };
