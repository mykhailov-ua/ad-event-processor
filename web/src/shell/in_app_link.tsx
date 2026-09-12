import type { MouseEvent, ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';

import { parseInAppLocation } from '@/lib/export_hub_return';

export type InAppLinkProps = {
  to: string;
  className?: string;
  children: ReactNode;
};

export function InAppLink({ to, className, children }: InAppLinkProps) {
  const navigate = useNavigate();

  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    if (event.defaultPrevented) {
      return;
    }
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button !== 0) {
      return;
    }
    event.preventDefault();
    navigate(parseInAppLocation(to));
  };

  return (
    <a className={className} href={to} onClick={handleClick}>
      {children}
    </a>
  );
}
