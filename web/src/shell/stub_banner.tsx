import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

type StubBannerProps = {
  title?: string;
  message: string;
};

export function StubBanner({ title = 'Not available', message, }: StubBannerProps) {
  return (
    <div  role="status">
      <p >{title}</p>
      <p >{message}</p>
    </div>
  );
}
