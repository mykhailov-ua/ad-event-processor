import { Button, type ButtonProps } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export function PrimaryActionButton({ className, shape = 'pill', ...props }: ButtonProps) {
  return <Button shape={shape} className={cn('px-4 text-sm', className)} {...props} />;
}

export function SecondaryActionButton({
  className,
  shape = 'pill',
  variant = 'outline',
  ...props
}: ButtonProps) {
  return (
    <Button
      shape={shape}
      variant={variant}
      className={cn('px-4 text-sm', className)}
      {...props}
    />
  );
}

export function FilterApplyButton({
  className,
  shape = 'pill',
  type = 'submit',
  ...props
}: ButtonProps) {
  return (
    <Button shape={shape} type={type} className={cn('text-sm', className)} {...props} />
  );
}

export function FilterResetButton({
  className,
  shape = 'pill',
  type = 'button',
  variant = 'outline',
  ...props
}: ButtonProps) {
  return (
    <Button
      shape={shape}
      type={type}
      variant={variant}
      className={cn('text-sm', className)}
      {...props}
    />
  );
}
