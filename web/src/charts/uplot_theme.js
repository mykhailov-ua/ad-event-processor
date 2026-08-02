/**
 * @returns {{ grid: string, line: string, text: string }}
 */
export function readUPlotTheme() {
  const style = getComputedStyle(document.documentElement);
  return {
    grid: style.getPropertyValue('--chart-grid').trim() || '#333',
    line: style.getPropertyValue('--chart-line').trim() || '#4a9eff',
    text: style.getPropertyValue('--text-secondary').trim() || '#999',
  };
}
