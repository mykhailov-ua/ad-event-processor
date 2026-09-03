#!/usr/bin/env node
/**
 * Replace space-separated admin-* tokens with Tailwind utilities.
 * Does not touch var(--admin-*) or partial substring matches inside other tokens.
 */
import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs';
import { join, extname } from 'node:path';
import { fileURLToPath } from 'node:url';

const SRC = join(fileURLToPath(new URL('.', import.meta.url)), '..', 'src');

const MAP = {
  'admin-campaigns-toolbar__filters-actions': 'flex shrink-0 flex-wrap items-center gap-2',
  'admin-campaigns-toolbar__filters-main': 'flex flex-1 flex-wrap items-center gap-2',
  'admin-campaigns-toolbar__filters':
    'mt-1 flex w-full flex-wrap items-center justify-between gap-x-3 gap-y-2 border-t border-zinc-200 pt-2 dark:border-zinc-800',
  'admin-campaigns-toolbar__summary': 'mt-1 border-t border-zinc-200 pt-2 dark:border-zinc-800',
  'admin-campaigns-toolbar__actions': 'p-0',
  'admin-campaigns-table-wrap': 'min-h-0 flex-1 overflow-auto',
  'admin-campaigns-toolbar': 'flex w-full flex-col',
  'admin-toolbar-row--filters': 'flex flex-wrap items-center gap-2',
  'admin-toolbar-row--actions': 'flex flex-wrap items-center gap-2',
  'admin-table-metric-conversion': 'tabular-nums font-bold text-indigo-600 dark:text-indigo-400',
  'admin-table-metric-secondary': 'tabular-nums text-zinc-600 dark:text-zinc-400',
  'admin-table-metric-primary': 'tabular-nums font-semibold text-zinc-900 dark:text-zinc-50',
  'admin-table-status-dot--active': 'inline-block h-2 w-2 rounded-full bg-green-500',
  'admin-table-status-dot--muted': 'inline-block h-2 w-2 rounded-full bg-zinc-400',
  'admin-table-wrap--static': 'overflow-hidden rounded-md',
  'admin-select-trigger--ghost':
    'h-8 border-0 bg-transparent shadow-none hover:bg-zinc-100 dark:hover:bg-zinc-800',
  'admin-row-status-edge--archived': '',
  'admin-row-status-edge--paused': '',
  'admin-row-status-edge--active': '',
  'admin-row-status-edge--unknown': '',
  'admin-section-nav--links': 'flex flex-wrap gap-1',
  'admin-label--filter-compact': 'flex items-center gap-1 text-xs',
  'admin-input--filter-compact': 'h-8 w-20 px-2 text-sm',
  'admin-import-section__note': 'text-sm text-zinc-500 dark:text-zinc-400',
  'admin-import-section__head': 'flex items-center justify-between gap-2',
  'admin-import-section__body': 'mt-2 flex flex-col gap-3',
  'admin-columns-menu__label': 'text-xs font-medium text-zinc-500 dark:text-zinc-400',
  'admin-columns-menu__item': 'flex items-center gap-2 text-sm',
  'admin-campaign-status__dot': 'h-1.5 w-1.5 shrink-0 rounded-full bg-current',
  'admin-campaign-status--paused': 'inline-flex items-center gap-1.5 text-xs text-amber-700 dark:text-amber-300',
  'admin-campaign-status--active': 'inline-flex items-center gap-1.5 text-xs text-green-700 dark:text-green-300',
  'admin-campaign-status--archived': 'inline-flex items-center gap-1.5 text-xs text-zinc-500',
  'admin-campaign-status': 'inline-flex items-center gap-1.5 text-xs',
  'admin-table-td--indicator': 'w-6 px-0.5 text-center',
  'admin-table-td--select': 'w-7 px-1 text-center',
  'admin-table-td--status': 'text-center',
  'admin-table-td--name': 'relative whitespace-normal py-1 pr-11 align-top',
  'admin-table-td--id': 'font-mono text-xs text-zinc-500 dark:text-zinc-400',
  'admin-table-th--indicator': 'w-6 px-0.5 text-center',
  'admin-table-th--select': 'w-7 px-1 text-center',
  'admin-table-th--status': 'text-center',
  'admin-table-th--id': 'font-mono text-xs',
  'admin-table--campaigns':
    'w-auto table-fixed border-collapse text-[13px] [&_th]:sticky [&_th]:top-0 [&_th]:z-[2] [&_th]:bg-zinc-50 [&_th]:px-3 [&_th]:py-1.5 [&_th]:text-xs [&_th]:font-semibold [&_th]:text-zinc-500 [&_td]:border-b [&_td]:border-zinc-100 [&_td]:px-3 [&_td]:py-1.5 dark:[&_th]:bg-zinc-900 dark:[&_th]:text-zinc-400 dark:[&_td]:border-zinc-800 [&_td.num]:text-right [&_tbody_tr:nth-child(even)_td]:bg-zinc-50/50 dark:[&_tbody_tr:nth-child(even)_td]:bg-zinc-900/40',
  'admin-profit-indicator--positive':
    'inline-flex h-[18px] w-[18px] items-center justify-center rounded bg-green-100 text-[11px] font-bold text-green-800 dark:bg-green-900/30 dark:text-green-400',
  'admin-profit-indicator--negative':
    'inline-flex h-[18px] w-[18px] items-center justify-center rounded bg-red-100 text-[11px] font-bold text-red-800 dark:bg-red-900/30 dark:text-red-400',
  'admin-profit-indicator--neutral':
    'inline-flex h-[18px] w-[18px] items-center justify-center rounded bg-zinc-100 text-[11px] font-bold text-zinc-400 dark:bg-zinc-800',
  'admin-profit-indicator': 'inline-flex select-none items-center justify-center',
  'admin-page-body--with-aside':
    'grid min-h-0 flex-1 grid-cols-1 gap-2 lg:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)]',
  'admin-header-breadcrumbs': 'min-w-0 overflow-hidden',
  'admin-header-actions': 'flex flex-wrap items-center gap-2',
  'admin-header-search': 'w-full max-w-md',
  'admin-header-center': 'flex w-full max-w-md min-w-0 justify-center justify-self-center',
  'admin-header-end': 'flex min-w-0 items-center justify-end justify-self-end gap-2',
  'admin-header-start': 'flex min-w-0 items-center gap-2',
  'admin-page-header-main': 'flex min-w-0 flex-wrap items-center gap-2',
  'admin-page-header': 'flex flex-wrap items-start justify-between gap-2',
  'admin-page-workspace':
    'flex min-h-0 flex-1 flex-col gap-2 rounded-md border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-page-footer':
    'relative z-[5] flex flex-wrap items-center gap-2 rounded-md border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-control-panel':
    'relative z-[5] rounded-md border border-zinc-200 bg-white p-2 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-page-main': 'flex min-h-0 min-w-0 flex-1 flex-col gap-2 overflow-hidden',
  'admin-page-aside': 'flex min-h-0 min-w-0 flex-col gap-2 overflow-auto',
  'admin-page-body': 'grid min-h-0 flex-1 grid-cols-1 gap-2',
  'admin-page-layout': 'flex min-h-0 flex-1 flex-col gap-2',
  'admin-footer-pagination': 'flex flex-wrap items-center gap-2',
  'admin-footer-bar': 'flex flex-wrap items-center gap-2',
  'admin-footer-exports': 'flex flex-wrap items-center gap-2',
  'admin-overlay-elevated':
    'rounded-lg border border-zinc-200 bg-white shadow-lg dark:border-zinc-800 dark:bg-zinc-950',
  'admin-metric-rate-warn': 'tabular-nums text-amber-600 dark:text-amber-400',
  'admin-metric-zero': 'tabular-nums text-zinc-400 dark:text-zinc-600',
  'admin-metric-fill-bar-segments__item': 'min-w-0 flex-1 self-end rounded-sm bg-blue-600/75',
  'admin-metric-fill-bar-segments': 'flex h-10 items-end gap-0.5 rounded bg-zinc-100 p-1 dark:bg-zinc-800/50',
  'admin-metric-fill-bar__fill': 'h-full rounded-full transition-[width] duration-200',
  'admin-metric-fill-bar__track': 'h-1.5 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-700',
  'admin-metric-fill-bar': 'flex flex-col gap-1',
  'admin-dashboard-bar-list__date': 'm-0 text-[11px] font-semibold text-zinc-500 dark:text-zinc-400',
  'admin-dashboard-bar-list__day': 'grid gap-2',
  'admin-dashboard-bar-list': 'grid max-h-[min(28rem,48vh)] gap-3 overflow-y-auto pr-0.5',
  'admin-dashboard-chart-legend': 'flex flex-wrap gap-3 text-xs',
  'admin-dashboard-chart-pane__title': 'text-xs font-semibold text-zinc-500 dark:text-zinc-400',
  'admin-dashboard-chart-pane': 'flex flex-col gap-2',
  'admin-dashboard-chart-empty': 'py-8 text-center text-sm text-zinc-500 dark:text-zinc-400',
  'admin-dashboard-chart-footnote': 'text-xs',
  'admin-dashboard-chart-shell': 'rounded-md border border-zinc-200 p-3 dark:border-zinc-800',
  'admin-dashboard-bar-shell': '',
  'admin-dashboard-kpi__value--positive': 'text-green-700 dark:text-green-400',
  'admin-dashboard-kpi__value--negative': 'text-red-700 dark:text-red-400',
  'admin-dashboard-kpi__value': 'text-lg font-semibold tabular-nums',
  'admin-dashboard-kpi__label': 'text-xs text-zinc-500 dark:text-zinc-400',
  'admin-dashboard-kpi--positive': 'border-green-200 dark:border-green-900',
  'admin-dashboard-kpi--negative': 'border-red-200 dark:border-red-900',
  'admin-dashboard-kpi--zero': 'opacity-80',
  'admin-dashboard-kpi--selected': 'ring-2 ring-blue-500',
  'admin-dashboard-kpi--clickable': 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900',
  'admin-dashboard-kpi': 'min-w-0 rounded-md border border-zinc-200 p-3 text-left dark:border-zinc-800',
  'admin-country-badge__flag': 'h-3 w-4 shrink-0',
  'admin-country-badge__code': 'truncate',
  'admin-country-badge':
    'inline-flex max-w-full items-center gap-0.5 overflow-hidden rounded border border-zinc-200 px-1 text-[10px] dark:border-zinc-700',
  'admin-country-badges': 'inline-flex max-w-full flex-nowrap items-center gap-0.5 overflow-hidden',
  'admin-filter-compact__label': 'text-xs text-zinc-500',
  'admin-filter-compact': 'flex items-center gap-1',
  'admin-filter-reset': 'text-sm',
  'admin-summary-sep': 'text-zinc-300 dark:text-zinc-600',
  'admin-toolbar-summary': 'flex flex-wrap items-center gap-2 text-sm',
  'admin-toolbar-group': 'flex flex-wrap items-center gap-1',
  'admin-toolbar-row': 'flex flex-wrap items-center gap-2',
  'admin-toolbar': 'flex flex-col gap-2',
  'admin-section-nav': 'flex flex-wrap gap-1',
  'admin-ops-block': 'rounded-md border border-zinc-200 p-3 dark:border-zinc-800',
  'admin-ops-block__title': 'text-sm font-semibold',
  'admin-ops-block__head': 'flex items-center justify-between gap-2',
  'admin-ops-stat-grid': 'grid gap-3 sm:grid-cols-2 lg:grid-cols-3',
  'admin-ops-stat-card': 'rounded-md border border-zinc-200 p-3 dark:border-zinc-800',
  'admin-ops-kv__label': 'text-zinc-500 dark:text-zinc-400',
  'admin-ops-kv__value': 'font-semibold tabular-nums',
  'admin-ops-kv__row': 'flex items-baseline justify-between gap-2 text-sm',
  'admin-ops-kv': 'grid gap-1',
  'admin-ops-status--ok': 'text-green-600 dark:text-green-400',
  'admin-ops-status--warn': 'text-amber-600 dark:text-amber-400',
  'admin-ops-status--fail': 'text-red-600 dark:text-red-400',
  'admin-ops-status--muted': 'text-zinc-500 dark:text-zinc-400',
  'admin-ops-empty': '',
  'admin-nav-sep': 'my-2 border-t border-zinc-200 dark:border-zinc-800',
  'admin-icon-btn':
    'inline-flex h-8 w-8 items-center justify-center rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800',
  'admin-btn--icon': 'h-8 w-8 p-0',
  'admin-btn--primary':
    'border border-zinc-900 bg-zinc-900 text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200',
  'admin-textarea':
    'min-h-[5rem] w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-950',
  'admin-select-trigger':
    'flex h-8 w-full items-center justify-between rounded-md border border-zinc-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950',
  'admin-select': 'relative w-full',
  'admin-menu-item':
    'relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-zinc-100 dark:hover:bg-zinc-800',
  'admin-input':
    'h-8 w-full rounded-md border border-zinc-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950',
  'admin-btn':
    'inline-flex h-8 items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm font-medium transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900',
  'admin-table-cell': 'tabular-nums',
  'admin-table-wrap': 'overflow-x-auto rounded-md border border-zinc-200 dark:border-zinc-800',
  'admin-table': 'w-full border-collapse text-sm',
  'admin-text-link': 'text-blue-600 hover:underline dark:text-blue-400',
  'admin-tabular-nums': 'tabular-nums',
  'admin-data-mono': 'font-mono tabular-nums',
  'admin-data-id': 'font-mono text-xs text-zinc-600 dark:text-zinc-400',
  'admin-fg-emphasis': 'font-semibold text-zinc-900 dark:text-zinc-50',
  'admin-fg-secondary': 'text-zinc-600 dark:text-zinc-400',
  'admin-stat-note': 'text-xs text-zinc-500 dark:text-zinc-400',
  'admin-section-title': 'text-sm font-semibold text-zinc-900 dark:text-zinc-100',
  'admin-field-row': 'grid grid-cols-[8rem_1fr] items-center gap-2',
  'admin-field': 'grid gap-1',
  'admin-label--stacked': 'flex flex-col gap-1 text-sm font-medium',
  'admin-label--range': 'flex flex-col gap-1',
  'admin-label': 'text-sm font-medium text-zinc-700 dark:text-zinc-300',
  'admin-stack--compact': 'flex flex-col gap-2',
  'admin-stack': 'flex flex-col gap-3',
  'admin-panel--raised':
    'rounded-md border border-zinc-200 bg-white p-3 shadow-sm dark:border-zinc-800 dark:bg-zinc-950',
  'admin-panel': 'rounded-md border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-chip':
    'inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-xs dark:border-zinc-700 dark:bg-zinc-900',
  'admin-muted': 'text-zinc-500 dark:text-zinc-400',
  'admin-positive': 'font-semibold text-green-700 dark:text-green-400',
  'admin-negative': 'font-semibold text-red-700 dark:text-red-400',
  'admin-row-positive': 'font-semibold text-green-700 dark:text-green-400',
  'admin-row-selected': 'bg-blue-50 dark:bg-blue-950/30',
  'admin-row-negative': '',
  'admin-row-warning': '',
  'admin-import-section': 'rounded-md border border-zinc-200 p-3 dark:border-zinc-800',
  'admin-import-panel': 'flex flex-col gap-4',
  'admin-import-panel__grid-2': 'grid gap-3 sm:grid-cols-2',
  'admin-import-panel__actions': 'flex flex-wrap gap-2',
  'admin-import-panel__job-row': 'flex flex-wrap items-center gap-2 text-sm',
  'admin-columns-menu': 'w-72',
  'admin-columns-menu__presets': 'flex flex-wrap gap-1 border-b border-zinc-200 p-2 dark:border-zinc-800',
  'admin-columns-menu__preset': 'text-xs text-blue-600 dark:text-blue-400',
  'admin-columns-menu__grid': 'grid max-h-64 grid-cols-2 gap-2 overflow-y-auto p-2',
  'admin-columns-menu__col': 'flex flex-col gap-1',
  'admin-columns-menu__title': 'text-xs font-semibold text-zinc-500',
  'admin-columns-menu__list': 'grid gap-0.5',
  'admin-columns-menu__footer': 'border-t border-zinc-200 p-2 dark:border-zinc-800',
  'admin-col-resize': 'absolute right-0 top-0 h-full w-1 cursor-col-resize',
  'admin-col-grip': 'absolute right-0 top-0 z-10 h-full w-2 cursor-col-resize',
  'admin-table-th-inner': 'flex items-center gap-1',
  'admin-page': 'flex min-h-0 flex-1 flex-col gap-2',
  'admin-content': 'flex min-h-0 flex-1 flex-col overflow-hidden p-3',
  'admin-header':
    'grid grid-cols-[minmax(0,1fr)_minmax(12rem,28rem)_minmax(0,1fr)] items-center gap-3 border-b border-zinc-200 bg-white px-3 py-2 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-main': 'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-zinc-50 dark:bg-zinc-950',
  'admin-nav-link':
    'relative block rounded px-2 py-1.5 no-underline hover:bg-zinc-100 dark:hover:bg-zinc-800 [&[aria-current=page]]:bg-blue-50 [&[aria-current=page]]:font-semibold [&[aria-current=page]]:text-blue-700 dark:[&[aria-current=page]]:bg-blue-950/40 dark:[&[aria-current=page]]:text-blue-300',
  'admin-sidebar':
    'flex w-48 shrink-0 flex-col gap-3 overflow-y-auto border-r border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-app-frame': 'flex min-h-screen flex-col',
  'admin-app': 'flex min-h-0 flex-1',
  'admin-dev-banner':
    'border-b border-amber-200 bg-amber-50 px-3 py-1 text-center text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200',
  'is-sidebar-collapsed': '',
  'admin-buyer-dashboard-toolbar': '',
  'admin-buyer-dashboard-toolbar__info': 'text-sm text-zinc-500 dark:text-zinc-400',
  'admin-preferences-dialog': '',
  'admin-preferences-dialog__header': '',
  'admin-preferences-dialog__body': '',
  'admin-preferences-dialog__footer': '',
  'admin-multi-select-trigger': '',
  'admin-multi-select-chevron': 'h-4 w-4 opacity-50',
  'admin-multi-select-menu': 'w-64',
  'admin-multi-select-menu__list': 'max-h-64 overflow-y-auto p-1',
  'admin-date-range-menu': 'rounded-md border border-zinc-200 bg-white p-0 shadow-lg dark:border-zinc-800 dark:bg-zinc-950',
  'admin-date-range-menu__body': 'p-3',
  'admin-date-range-menu__footer': 'flex justify-end gap-2 border-t border-zinc-200 p-2 dark:border-zinc-800',
  'admin-select-content': 'z-50 max-h-96 overflow-hidden',
  'admin-select-item': '',
  'admin-select-separator': 'my-1 h-px bg-zinc-200 dark:bg-zinc-800',
  'admin-dropdown-content': 'z-50',
  'admin-dropdown-separator': 'my-1 h-px bg-zinc-200 dark:bg-zinc-800',
  'admin-error-details': 'mt-4 rounded-md border border-zinc-200 dark:border-zinc-800',
  'admin-error-details__header': 'border-b border-zinc-200 px-3 py-2 dark:border-zinc-800',
  'admin-error-details__title': 'text-xs font-semibold text-zinc-500',
  'admin-error-details__body': 'max-h-48 overflow-auto p-3 text-xs font-mono',
  'admin-error-page': 'flex min-h-0 flex-1 items-center justify-center p-6',
  'admin-error-page--standalone': 'min-h-screen bg-zinc-50 dark:bg-zinc-950',
  'admin-error-page__card': 'w-full max-w-lg rounded-lg border border-zinc-200 bg-white p-6 dark:border-zinc-800 dark:bg-zinc-950',
  'admin-error-page__eyebrow': 'text-xs font-semibold uppercase tracking-wide text-zinc-500',
  'admin-error-page__title': 'mt-2 text-xl font-semibold',
  'admin-error-page__message': 'mt-2 text-sm text-zinc-600 dark:text-zinc-400',
  'admin-error-page__route': 'mt-2 font-mono text-xs text-zinc-500',
  'admin-error-page__actions': 'mt-6 flex flex-wrap gap-2',
  'admin-table-name': 'flex flex-col gap-0.5',
  'admin-table-name__countries': 'text-xs text-zinc-500',
  'admin-table-name-link': 'font-medium hover:underline',
  'admin-table-row-action': 'absolute right-2 top-1/2 -translate-y-1/2',
  'admin-status-links': 'flex flex-wrap gap-2',
  'admin-table-td--truncate': 'max-w-0 truncate',
  'admin-table-td--actions': 'w-10 text-center',
  'admin-toolbar-row--sections': 'flex flex-wrap items-center gap-2 border-t border-zinc-200 pt-2 dark:border-zinc-800',
  'admin-country-badge--more': 'text-[10px] text-zinc-500',
  'admin-reset-dialog': 'max-w-md',
  'admin-reset-dialog__header': 'space-y-1',
  'admin-reset-list': 'list-disc space-y-1 pl-5 text-sm',
  'admin-reset-dialog__footer': 'flex justify-end gap-2',
  'is-active': 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900',
  'is-open': 'rotate-180',
};

function migrateTokens(classString) {
  if (!classString.trim()) {
    return classString;
  }
  const parts = classString.split(/\s+/).filter(Boolean);
  const out = [];
  for (const part of parts) {
    if (part.startsWith('var(--admin-') || part.includes('[var(--admin-')) {
      out.push(part);
      continue;
    }
    const mapped = MAP[part];
    if (mapped === undefined) {
      if (part.startsWith('admin-')) {
        console.warn(`unmapped: ${part}`);
      }
      out.push(part);
    } else if (mapped) {
      out.push(mapped);
    }
  }
  return out.join(' ').replace(/\s+/g, ' ').trim();
}

function migrateContent(content) {
  let out = content;
  out = out.replace(/className="([^"]*)"/g, (_, classes) => `className="${migrateTokens(classes)}"`);
  out = out.replace(/className='([^']*)'/g, (_, classes) => `className='${migrateTokens(classes)}'`);
  out = out.replace(/cn\(([^)]*)\)/g, (match, inner) => {
    const migrated = inner
      .replace(/'([^']*)'/g, (_, s) => `'${migrateTokens(s)}'`)
      .replace(/"([^"]*)"/g, (_, s) => `"${migrateTokens(s)}"`);
    return `cn(${migrated})`;
  });
  out = out.replace(/return '([^']+)'/g, (match, classes) => {
    if (!classes.includes('admin-')) {
      return match;
    }
    return `return '${migrateTokens(classes)}'`;
  });
  return out;
}

function walk(dir, files = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      if (name === 'styles') continue;
      walk(p, files);
    } else if (['.ts', '.tsx'].includes(extname(name))) {
      files.push(p);
    }
  }
  return files;
}

const files = walk(SRC);
let changed = 0;
for (const file of files) {
  const before = readFileSync(file, 'utf8');
  const after = migrateContent(before);
  if (after !== before) {
    writeFileSync(file, after, 'utf8');
    changed++;
  }
}

console.log(`migrate_admin_classes: updated ${changed} files`);
