import type { ComponentProps } from 'react';
import { createPortal } from 'react-dom';
import { Toaster as Sonner } from 'sonner';

import { useTheme } from '@/hooks/use_theme';
import { cn } from '@/lib/utils';

type ToasterProps = ComponentProps<typeof Sonner>;

const TOAST_DURATION_MS = 3000;

const toastShellClass = cn(
  'flex w-[var(--width)] items-center gap-2 rounded-[8px] border px-3 py-1.5 shadow-none'
);

const toastClassNames = {
  toast: toastShellClass,
  title: 'text-[13px] leading-[18px] font-medium text-foreground',
  description: 'text-[13px] leading-[18px] font-normal text-admin-fg-secondary',
  closeButton: 'hidden',
  error:
    'border-destructive/35 bg-destructive/16 text-foreground [&_[data-icon]]:text-destructive',
  warning:
    'border-admin-warn-border/45 bg-admin-warn-bg/55 text-foreground [&_[data-icon]]:text-admin-warn-fg',
  success:
    'border-admin-status-active/35 bg-admin-status-active/16 text-foreground [&_[data-icon]]:text-admin-positive-fg',
  info:
    'border-primary/35 bg-primary/18 text-foreground [&_[data-icon]]:text-admin-metric-conversion-fg',
  default:
    'border-primary/35 bg-primary/18 text-foreground [&_[data-icon]]:text-admin-metric-conversion-fg',
};

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return createPortal(
    <Sonner
      className="toaster"
      closeButton={false}
      cn={cn}
      duration={TOAST_DURATION_MS}
      expand={false}
      gap={8}
      offset="1rem"
      position="bottom-right"
      theme={theme}
      visibleToasts={4}
      toastOptions={{
        duration: TOAST_DURATION_MS,
        unstyled: true,
        classNames: toastClassNames,
      }}
      {...props}
    />,
    document.body
  );
};

export { Toaster };
