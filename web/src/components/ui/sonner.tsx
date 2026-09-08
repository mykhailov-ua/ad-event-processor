import type { ComponentProps } from 'react';
import { createPortal } from 'react-dom';
import { Toaster as Sonner } from 'sonner';

import { useTheme } from '@/hooks/use_theme';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

type ToasterProps = ComponentProps<typeof Sonner>;

const TOAST_DURATION_MS = 3000;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return createPortal(
    <Sonner
      closeButton
      duration={TOAST_DURATION_MS}
      expand={false}
      gap={10}
      offset="1rem"
      position="bottom-right"
      theme={theme}
      visibleToasts={4}
      className="toaster group"
      toastOptions={{
        duration: TOAST_DURATION_MS,
        classNames: {
          toast: cn(
            'group toast group-[.toaster]:rounded-sm group-[.toaster]:border group-[.toaster]:px-4 group-[.toaster]:py-3 group-[.toaster]:pr-10 group-[.toaster]:text-[13px]',
            adminKit.toastSurface
          ),
          success:
            'group-[.toaster]:border-admin-status-active/30 group-[.toaster]:bg-admin-status-active/15 group-[.toaster]:text-admin-positive',
          error: cn('group-[.toaster]:text-destructive', adminKit.errorSurface),
          warning:
            'group-[.toaster]:border-admin-warn-border/50 group-[.toaster]:bg-admin-warn-bg/80 group-[.toaster]:text-admin-warn',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton:
            'group-[.toast]:rounded-sm group-[.toast]:border group-[.toast]:border-border group-[.toast]:bg-card/90 group-[.toast]:text-foreground',
          cancelButton: 'group-[.toast]:text-muted-foreground',
          closeButton:
            'group-[.toast]:border-border/60 group-[.toast]:bg-card/90 group-[.toast]:text-muted-foreground group-[.toast]:opacity-100 hover:group-[.toast]:bg-card',
        },
      }}
      {...props}
    />,
    document.body
  );
};

export { Toaster };
