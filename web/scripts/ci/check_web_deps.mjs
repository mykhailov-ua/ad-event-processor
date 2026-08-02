import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const pkgPath = resolve(ROOT, 'package.json');
const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'));

const forbidden = [
  'react',
  'react-dom',
  'react-router',
  'react-router-dom',
  'vite',
  '@vitejs/plugin-react',
  'chart.js',
  'recharts',
  'redux',
  'zustand',
  'axios',
];

const deps = { ...pkg.dependencies, ...pkg.devDependencies };
const hits = forbidden.filter((name) => deps[name]);
if (hits.length > 0) {
  console.error('Forbidden packages in package.json:', hits.join(', '));
  process.exit(1);
}

const allowedProd = ['uplot'];
const prod = Object.keys(pkg.dependencies ?? {});
const extra = prod.filter((d) => !allowedProd.includes(d));
if (extra.length > 0) {
  console.error('Unexpected production dependencies:', extra.join(', '));
  process.exit(1);
}

console.log('web deps check: OK');
