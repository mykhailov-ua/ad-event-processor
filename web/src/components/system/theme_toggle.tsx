import { Moon, Sun } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { themeToggleLabel } from '@/lib/theme';
import { cn } from '@/lib/utils';
import { useTheme } from '@/providers/theme_provider';

export type ThemeToggleProps = {
  className?: string;
  showLabel?: boolean;
};

export function ThemeToggle({ className, showLabel = false }: ThemeToggleProps) {
  const { theme, setTheme } = useTheme();
  const nextTheme = theme === 'dark' ? 'light' : 'dark';
  const Icon = theme === 'dark' ? Moon : Sun;

  const button = (
    <Button
      aria-label={themeToggleLabel(theme)}
      className={cn(showLabel ? 'gap-2 rounded-full' : 'size-9 rounded-full p-0', className)}
      onClick={() => setTheme(nextTheme)}
      size={showLabel ? 'sm' : 'icon'}
      type="button"
      variant="ghost"
    >
      <Icon className="h-4 w-4" />
      {showLabel ? <span>{theme === 'dark' ? 'Dark' : 'Light'}</span> : null}
    </Button>
  );

  if (showLabel) {
    return button;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{button}</TooltipTrigger>
      <TooltipContent side="right">{themeToggleLabel(theme)}</TooltipContent>
    </Tooltip>
  );
}
