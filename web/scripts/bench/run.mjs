#!/usr/bin/env node

import { performance } from 'node:perf_hooks';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const SRC = join(ROOT, 'src');

const { cn } = await import(pathToFileURL(join(SRC, 'lib/cn.js')).href);

const iterations = 50_000;
const start = performance.now();
for (let i = 0; i < iterations; i += 1) {
  cn('a', i % 2 === 0 ? 'b' : false, 'c');
}
const elapsed = performance.now() - start;
console.log(`cn bench: ${iterations} ops in ${elapsed.toFixed(2)}ms`);
