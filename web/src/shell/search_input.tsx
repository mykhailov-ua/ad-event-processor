import { Search } from 'lucide-react';
import type { ComponentProps } from 'react';

import { Input } from '@/components/ui/input';
import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

export type SearchInputProps = Omit<ComponentProps<'input'>, 'type'> & {
  className?: string;
  wrapperClassName?: string;
};

export function SearchInput({ className, wrapperClassName, ...props }: SearchInputProps) {
  return (
    <div className={cn(adminChrome.controlFieldGroup, 'min-w-0', wrapperClassName)}>
      <Search aria-hidden className="size-3.5 shrink-0 text-muted-foreground" />
      <Input
        className={cn(adminChrome.controlFieldInset, 'w-full', className)}
        type="search"
        {...props}
      />
    </div>
  );
}
