import { TelegramBotsDirectory } from '@/domains/telegram/bots_directory';
import { useTelegramBotsPageWorkspace } from '@/domains/telegram/use_telegram_bots_page_workspace';

export function TelegramBotsPage() {
  return <TelegramBotsDirectory {...useTelegramBotsPageWorkspace()} />;
}
