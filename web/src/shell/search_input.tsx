import { Search } from 'lucide-react';
import type { ComponentProps } from 'react';

import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

export type SearchInputProps = Omit<ComponentProps<'input'>, 'type'> & {
  wrapperClassName?: string;
};

export function SearchInput({ className, wrapperClassName, ...props }: SearchInputProps) {
  return (
    <div className={cn('relative flex min-w-0 items-center', wrapperClassName)}>
      <Search
        aria-hidden
        className="pointer-events-none absolute left-2.5 h-3.5 w-3.5 text-muted-foreground"
      />
      <Input className={cn('bg-background pl-8 text-foreground', className)} type="search" {...props} />
    </div>
  );
}
