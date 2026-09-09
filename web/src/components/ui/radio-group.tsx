import * as React from 'react';

import { Label } from '@/components/ui/label';
import { adminKit } from '@/lib/admin_kit';
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
    { value,
      defaultValue,
      onValueChange,
      name: nameProp,
      disabled = false,
      children,
      ...props
    },
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
        <div
          ref={ref}
         
          role="radiogroup"
          {...props}
        >
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
  ({ value, id, disabled: itemDisabled, children, ...props }, ref) => {
    const { name, value: selectedValue, onValueChange, disabled: groupDisabled } =
      useRadioGroupContext();
    const disabled = groupDisabled || itemDisabled;
    const checked = selectedValue === value;
    const inputId = id ?? `${name}-${value}`;

    return (
      <div >
        <span >
          <input
            ref={ref}
            checked={checked}
           
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
           
          >
            {checked ? <span  /> : null}
          </span>
        </span>
        {children ? (
          <Label  htmlFor={inputId}>
            {children}
          </Label>
        ) : null}
      </div>
    );
  }
);
RadioGroupItem.displayName = 'RadioGroupItem';

export { RadioGroup, RadioGroupItem };
