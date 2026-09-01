import { ApiError } from '@/api/client';
import { ErrorBlock } from '@/components/system/error_block';
import { SectionNav } from '@/components/system/section_nav';
import { StubBanner } from '@/components/system/stub_banner';
import type { SectionNavItem } from '@/lib/nav_config';

export const TELEGRAM_NAV_ITEMS: SectionNavItem[] = [
  { path: '/telegram/bots', label: 'Bots', exact: true },
  { path: '/telegram/postbacks', label: 'Postbacks' },
];

export function TelegramNav() {
  return <SectionNav items={TELEGRAM_NAV_ITEMS} label="Telegram sections" />;
}

export function telegramPanelError(error: Error, title: string) {
  if (error instanceof ApiError && error.status === 403) {
    return <StubBanner title={`${title} forbidden`} message={error.message} />;
  }
  if (error instanceof ApiError && error.status === 501) {
    return <StubBanner title={`${title} unavailable`} message={error.message} />;
  }
  return <ErrorBlock title={title} message={error.message} />;
}
