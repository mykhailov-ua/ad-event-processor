import { Search } from 'lucide-react';
import type { ComponentProps } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type SearchInputProps = Omit<ComponentProps<'input'>, 'type'> & {
};

export function SearchInput({ ...props }: SearchInputProps) {
  return (
    <div >
      <Search aria-hidden  />
      <input
       
        type="search"
        {...props}
      />
    </div>
  );
}
