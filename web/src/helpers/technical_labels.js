import { devModeEnabled } from './dev_mode.js';

/**
 * @param {string|number} n
 * @returns {string}
 */
function formatNum(n) {
  const v = Number(n);
  if (!Number.isFinite(v)) return String(n);
  return v.toLocaleString('en-US');
}

/** @type {Array<{ re: RegExp, fmt: (m: RegExpMatchArray) => string }>} */
const DETAIL_PATTERNS = [
  {
    re: /^file-max=(\d+) somaxconn=(\d+)$/,
    fmt: (m) => `System limits OK — max open files: ${formatNum(m[1])}, TCP listen backlog: ${formatNum(m[2])}`,
  },
  {
    re: /^fs\.file-max=(\d+) want >= (\d+)$/,
    fmt: (m) => `Max open files too low (${formatNum(m[1])}; recommended at least ${formatNum(m[2])})`,
  },
  {
    re: /^net\.core\.somaxconn=(\d+) want >= (\d+)$/,
    fmt: (m) => `TCP listen backlog too small (${formatNum(m[1])}; recommended at least ${formatNum(m[2])})`,
  },
  {
    re: /^CONFIG_XDP enabled$/,
    fmt: () => 'Kernel XDP support is enabled',
  },
  {
    re: /^bpf syscall available$/,
    fmt: () => 'BPF syscall is available',
  },
  {
    re: /^bpf syscall unavailable$/,
    fmt: () => 'BPF syscall is not available — edge XDP may not work',
  },
  {
    re: /^linux only$/,
    fmt: () => 'Linux-only check (skipped on this host)',
  },
];

/**
 * Humanize doctor / ops technical detail strings unless dev mode is on.
 *
 * @param {string|null|undefined} text
 * @returns {string|null|undefined}
 */
export function humanizeTechnicalDetail(text) {
  if (text == null || text === '') return text;
  if (devModeEnabled()) return text;
  const trimmed = String(text).trim();
  for (let i = 0; i < DETAIL_PATTERNS.length; i++) {
    const match = trimmed.match(DETAIL_PATTERNS[i].re);
    if (match) return DETAIL_PATTERNS[i].fmt(match);
  }
  return trimmed;
}
