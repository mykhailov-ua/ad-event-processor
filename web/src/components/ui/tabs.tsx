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

const TabsList = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      role="tablist"
      className={cn(
        'inline-flex h-9 items-center rounded-sm bg-zinc-100 p-1 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400',
        className,
      )}
      {...props}
    />
  ),
);
TabsList.displayName = 'TabsList';

const TabsTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & { value: string }
>(({ className, value, onClick, ...props }, ref) => {
  const { active, setActive } = useTabsContext();
  const selected = active === value;

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={selected}
      className={cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1 text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
        selected
          ? 'bg-white text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100'
          : 'text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100',
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
