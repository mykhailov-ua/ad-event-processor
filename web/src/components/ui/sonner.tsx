import type { ComponentProps } from 'react';
import { Toaster as Sonner } from 'sonner';

import { useTheme } from '@/hooks/use_theme';

type ToasterProps = ComponentProps<typeof Sonner>;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return (
    <Sonner
      theme={theme}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            'group toast group-[.toaster]:rounded-sm group-[.toaster]:border group-[.toaster]:border-border group-[.toaster]:bg-card group-[.toaster]:text-card-foreground group-[.toaster]:px-4 group-[.toaster]:py-3 group-[.toaster]:text-[13px] group-[.toaster]:shadow-md',
          success:
            'group-[.toaster]:border-admin-status-active/30 group-[.toaster]:bg-admin-status-active/10 group-[.toaster]:text-admin-positive',
          error:
            'group-[.toaster]:border-destructive/30 group-[.toaster]:bg-destructive/10 group-[.toaster]:text-destructive',
          warning:
            'group-[.toaster]:border-admin-warn-border group-[.toaster]:bg-admin-warn-bg group-[.toaster]:text-admin-warn',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton:
            'group-[.toast]:rounded-sm group-[.toast]:border group-[.toast]:border-border group-[.toast]:bg-background group-[.toast]:text-foreground',
          cancelButton: 'group-[.toast]:text-muted-foreground',
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
