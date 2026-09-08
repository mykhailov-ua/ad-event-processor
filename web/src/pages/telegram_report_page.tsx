import {
  TELEGRAM_REPORT_CONFIGS,
  type TelegramReportKey,
} from '@/domains/reports/telegram_report_meta';
import { TelegramReportDirectory } from '@/domains/reports/telegram_report_directory';
import { useTelegramReportWorkspace } from '@/domains/reports/use_telegram_report_workspace';

function TelegramReportPageBody<K extends TelegramReportKey>({ reportKey }: { reportKey: K }) {
  const config = TELEGRAM_REPORT_CONFIGS[reportKey];
  const workspace = useTelegramReportWorkspace(config.fetch, config.enableExport);
  return (
    <TelegramReportDirectory config={config} {...workspace} enableExport={config.enableExport} />
  );
}

export function TelegramReportPage({ reportKey }: { reportKey: TelegramReportKey }) {
  return <TelegramReportPageBody reportKey={reportKey} />;
}
