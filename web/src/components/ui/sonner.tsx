import type { ComponentProps } from 'react';
import { Toaster as Sonner } from 'sonner';

import { useTheme } from '@/hooks/use_theme';

type ToasterProps = ComponentProps<typeof Sonner>;

const TOAST_DURATION_MS = 3000;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme } = useTheme();

  return (
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
          toast:
            'group toast group-[.toaster]:rounded-sm group-[.toaster]:border group-[.toaster]:border-border/50 group-[.toaster]:bg-card/80 group-[.toaster]:text-card-foreground group-[.toaster]:backdrop-blur-sm group-[.toaster]:px-4 group-[.toaster]:py-3 group-[.toaster]:pr-10 group-[.toaster]:text-[13px] group-[.toaster]:shadow-md group-[.toaster]:shadow-black/10',
          success:
            'group-[.toaster]:border-admin-status-active/25 group-[.toaster]:bg-admin-status-active/15 group-[.toaster]:text-admin-positive',
          error:
            'group-[.toaster]:border-destructive/25 group-[.toaster]:bg-destructive/15 group-[.toaster]:text-destructive',
          warning:
            'group-[.toaster]:border-admin-warn-border/60 group-[.toaster]:bg-admin-warn-bg/75 group-[.toaster]:text-admin-warn',
          description: 'group-[.toast]:text-muted-foreground',
          actionButton:
            'group-[.toast]:rounded-sm group-[.toast]:border group-[.toast]:border-border group-[.toast]:bg-background/80 group-[.toast]:text-foreground',
          cancelButton: 'group-[.toast]:text-muted-foreground',
          closeButton:
            'group-[.toast]:border-border/50 group-[.toast]:bg-background/70 group-[.toast]:text-muted-foreground group-[.toast]:opacity-100 hover:group-[.toast]:bg-background/90',
        },
      }}
      {...props}
    />
  );
};

export { Toaster };
