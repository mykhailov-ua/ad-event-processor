import * as React from 'react';

import { Label } from '@/components/ui/label';
import { adminKit, adminSpacing, adminTypography } from '@/lib/admin_kit';
import { useControllableState } from '@/lib/controllable_state';
import { cn } from '@/lib/utils';

type RadioGroupContextValue = {
  name: string;
  value?: string;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
};

const RadioGroupContext = React.createContext<RadioGroupContextValue | null>(null);

function useRadioGroupContext() {
  const context = React.useContext(RadioGroupContext);
  if (!context) {
    throw new Error('RadioGroupItem must be used within <RadioGroup>');
  }
  return context;
}

export type RadioGroupProps = Omit<React.HTMLAttributes<HTMLDivElement>, 'onChange'> & {
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  name?: string;
  disabled?: boolean;
};

const RadioGroup = React.forwardRef<HTMLDivElement, RadioGroupProps>(
  (
    { value, defaultValue, onValueChange, name: nameProp, disabled = false, className, children, ...props },
    ref
  ) => {
    const [internalValue, setInternalValue] = useControllableState({
      value,
      defaultValue,
      onChange: onValueChange,
    });
    const name = React.useId();
    const groupName = nameProp ?? name;

    const contextValue = React.useMemo(
      () => ({
        name: groupName,
        value: internalValue,
        onValueChange: setInternalValue,
        disabled,
      }),
      [groupName, internalValue, setInternalValue, disabled]
    );

    return (
      <RadioGroupContext.Provider value={contextValue}>
        <div ref={ref} className={cn(`grid ${adminSpacing.gap.md}`, className)} role="radiogroup" {...props}>
          {children}
        </div>
      </RadioGroupContext.Provider>
    );
  }
);
RadioGroup.displayName = 'RadioGroup';

export type RadioGroupItemProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  'type' | 'checked' | 'onChange'
> & {
  value: string;
};

const RadioGroupItem = React.forwardRef<HTMLInputElement, RadioGroupItemProps>(
  ({ value, id, disabled: itemDisabled, className, children, ...props }, ref) => {
    const { name, value: selectedValue, onValueChange, disabled: groupDisabled } =
      useRadioGroupContext();
    const disabled = groupDisabled || itemDisabled;
    const checked = selectedValue === value;
    const inputId = id ?? `${name}-${value}`;

    return (
      <div className={cn('flex items-center', adminSpacing.gap.sm, className)}>
        <span className="relative inline-flex h-4 w-4 shrink-0">
          <input
            ref={ref}
            checked={checked}
            className="peer absolute inset-0 z-10 h-full w-full cursor-pointer opacity-0"
            disabled={disabled}
            id={inputId}
            name={name}
            type="radio"
            value={value}
            onChange={() => onValueChange?.(value)}
            {...props}
          />
          <span
            aria-hidden
            className={cn(
              'pointer-events-none flex h-4 w-4 items-center justify-center border border-muted-foreground/70 bg-card transition-colors peer-focus-visible:outline-none peer-focus-visible:ring-2 peer-focus-visible:ring-primary/30 peer-disabled:cursor-not-allowed peer-disabled:border-input peer-disabled:bg-admin-input-disabled peer-disabled:opacity-100 peer-checked:border-primary',
              adminKit.controlRadius
            )}
          >
            {checked ? <span className="h-2 w-2 rounded-full bg-primary" /> : null}
          </span>
        </span>
        {children ? (
          <Label className={adminTypography.body} htmlFor={inputId}>
            {children}
          </Label>
        ) : null}
      </div>
    );
  }
);
RadioGroupItem.displayName = 'RadioGroupItem';

export { RadioGroup, RadioGroupItem };
