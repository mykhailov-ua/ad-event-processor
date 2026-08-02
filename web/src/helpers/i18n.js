/** @type {Record<string, string>} */
const RU = {
  'nav.overview': 'Обзор',
  'nav.campaigns': 'Кампании',
  'nav.portfolio': 'Портфель',
  'nav.billing': 'Биллинг',
  'nav.reports': 'Отчёты',
  'nav.ops': 'Операции',
  'nav.settings': 'Настройки',
  'action.load': 'Загрузить',
  'action.export': 'Экспорт CSV',
  'status.loading': 'Загрузка…',
  'report.compare': 'Сравнить с прошлым периодом',
};

/**
 * Resolve a UI string (v1 Russian catalog).
 *
 * @param {string} key
 * @param {string} [fallback]
 * @returns {string}
 */
export function t(key, fallback = key) {
  return RU[key] ?? fallback;
}
