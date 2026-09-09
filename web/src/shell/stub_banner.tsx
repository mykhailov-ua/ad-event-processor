import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

type StubBannerProps = {
  title?: string;
  message: string;
  className?: string;
};

export function StubBanner({ title = 'Not available', message, className }: StubBannerProps) {
  return (
    <div className={cn(uiSurfaces.messageMuted, className)} role="status">
      <p className="m-0 text-base font-semibold">{title}</p>
      <p className="m-0 text-sm">{message}</p>
    </div>
  );
}
