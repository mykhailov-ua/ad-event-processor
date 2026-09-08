import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

/** Default class-B list body: capped height, own scrollbar, no page scroll bleed (VL-13b). */
export const DROPDOWN_MENU_SCROLL_BODY_CLASS = cn(
  adminChrome.menuList,
  'ui-scrollbar max-h-60 overflow-y-auto overscroll-y-contain'
);
