import { Search } from 'lucide-react';
import type { ComponentProps } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type SearchInputProps = Omit<ComponentProps<'input'>, 'type'> & {
  wrapperClassName?: string;
};

export function SearchInput({ className, wrapperClassName, ...props }: SearchInputProps) {
  return (
    <div className={cn(adminChrome.controlFieldGroup, 'min-w-0', wrapperClassName)}>
      <Search aria-hidden className="size-3.5 shrink-0 text-muted-foreground" />
      <input
        className={cn(adminChrome.controlFieldInset, adminKit.controlText, className)}
        type="search"
        {...props}
      />
    </div>
  );
}
