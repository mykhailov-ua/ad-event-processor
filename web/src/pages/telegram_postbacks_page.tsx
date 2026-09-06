import { TelegramPostbacksDirectory } from '@/domains/telegram/postbacks_directory';
import { useTelegramPostbacksPageWorkspace } from '@/domains/telegram/use_telegram_postbacks_page_workspace';

export function TelegramPostbacksPage() {
  return <TelegramPostbacksDirectory {...useTelegramPostbacksPageWorkspace()} />;
}
