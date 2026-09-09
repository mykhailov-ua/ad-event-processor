import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export function FormSectionLabel({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <p >
      {children}
    </p>
  );
}

export function InputWithIcon({
  icon: Icon,
  ...props
}: React.ComponentProps<'input'> & { icon: LucideIcon }) {
  return (
    <div >
      <Icon aria-hidden  strokeWidth={2} />
      <input
       
        {...props}
      />
    </div>
  );
}

export function DashedActionZone({
  children,
  onClick,
  type = 'button',
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: 'button' | 'submit';
}) {
  return (
    <button
     
      onClick={onClick}
      type={type}
    >
      {children}
    </button>
  );
}
