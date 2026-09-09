import * as React from 'react';

import { useControllableState } from '@/lib/controllable_state';
import { cn } from '@/lib/utils';
import { adminKit } from '@/lib/admin_kit';

type TabsProps = {
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  children?: React.ReactNode;
};

function Tabs({
  value,
  defaultValue,
  onValueChange,
  children,
}: TabsProps) {
  const [active, setActive] = useControllableState({
    value,
    defaultValue,
    onChange: onValueChange,
  });

  return (
    <TabsContext.Provider value={{ active: active ?? '', setActive }}>
      <div >{children}</div>
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
>(({ variant = 'segmented', ...props }, ref) => (
  <div
    ref={ref}
    role="tablist"
   
    {...props}
  />
));
TabsList.displayName = 'TabsList';

const TabsTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    value: string;
    variant?: 'segmented' | 'pill' | 'underline';
  }
>(({ value, variant = 'segmented', onClick, ...props }, ref) => {
  const { active, setActive } = useTabsContext();
  const selected = active === value;

  return (
    <button
      ref={ref}
      type="button"
      role="tab"
      aria-selected={selected}
     
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
>(({ value, ...props }, ref) => {
  const { active } = useTabsContext();
  if (active !== value) {
    return null;
  }

  return <div ref={ref} role="tabpanel" {...props} />;
});
TabsContent.displayName = 'TabsContent';

export { Tabs, TabsList, TabsTrigger, TabsContent };
