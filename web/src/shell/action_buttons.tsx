import { Button, type ButtonProps } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export function PrimaryActionButton({ shape = 'default',
  variant = 'brand',
  ...props
}: ButtonProps) {
  return <Button shape={shape} variant={variant} {...props} />;
}

export function SecondaryActionButton({ shape = 'pill',
  variant = 'outline',
  ...props
}: ButtonProps) {
  return <Button shape={shape} variant={variant} {...props} />;
}

export function FilterApplyButton({ shape = 'pill',
  type = 'submit',
  variant = 'brand',
  children = 'Apply',
  ...props
}: ButtonProps) {
  return (
    <Button shape={shape} type={type} variant={variant} {...props}>
      {children}
    </Button>
  );
}

export function FilterResetButton({ shape = 'pill',
  type = 'button',
  variant = 'outline',
  ...props
}: ButtonProps) {
  return (
    <Button shape={shape} type={type} variant={variant} {...props} />
  );
}
