import { Moon, Sun } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { themeToggleLabel } from '@/lib/theme';
import { cn } from '@/lib/utils';
import { useTheme } from '@/hooks/use_theme';

export type ThemeToggleProps = {
  showLabel?: boolean;
};

export function ThemeToggle({ showLabel = false }: ThemeToggleProps) {
  const { theme, setTheme } = useTheme();
  const nextTheme = theme === 'dark' ? 'light' : 'dark';
  const Icon = theme === 'dark' ? Sun : Moon;

  const button = (
    <Button
      aria-label={themeToggleLabel(theme)}
      className={cn(!showLabel && 'size-7 p-0')}
      type="button"
      variant="outline"
      onClick={() => setTheme(nextTheme)}
    >
      <Icon aria-hidden className="h-4 w-4" />
      {showLabel ? <span>{theme === 'dark' ? 'Light' : 'Dark'}</span> : null}
    </Button>
  );

  if (showLabel) {
    return button;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>{button}</TooltipTrigger>
      <TooltipContent align="end" side="top">
        {themeToggleLabel(theme)}
      </TooltipContent>
    </Tooltip>
  );
}
