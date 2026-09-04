import { Search } from 'lucide-react';
import type { ComponentProps } from 'react';

import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

export type SearchInputProps = Omit<ComponentProps<'input'>, 'type'> & {
  wrapperClassName?: string;
};

export function SearchInput({ className, wrapperClassName, ...props }: SearchInputProps) {
  return (
    <div className={cn('admin-search-input', wrapperClassName)}>
      <Search aria-hidden className="admin-search-input__icon" />
      <Input className={cn('admin-search-input__field', className)} type="search" {...props} />
    </div>
  );
}
