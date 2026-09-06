import { TelegramBotEditor } from '@/domains/telegram/bot_editor';
import { useTelegramBotEditorPageWorkspace } from '@/domains/telegram/use_telegram_bot_editor_page_workspace';

export function TelegramBotEditorPage() {
  return <TelegramBotEditor {...useTelegramBotEditorPageWorkspace()} />;
}
