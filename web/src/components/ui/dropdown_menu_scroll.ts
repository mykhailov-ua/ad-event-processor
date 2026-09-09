import { adminChrome } from '@/lib/admin_chrome';
import { uiScrollbarClass } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

/** Default class-B list body: capped height, own scrollbar, no page scroll bleed (VL-13b). */
export const DROPDOWN_MENU_SCROLL_BODY_CLASS = cn(
  adminChrome.menuList,
  uiScrollbarClass,
  'max-h-60 overflow-y-auto overscroll-y-contain'
);
