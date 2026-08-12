/**
 * Node custom loader: resolve foo.js → foo.ts when the TypeScript source exists.
 * Used by unit tests and micro-benchmarks that import source modules directly.
 */
import { existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

/**
 * @param {string} specifier
 * @param {{ parentURL?: string }} context
 * @param {(s: string, c: object) => Promise<object>} nextResolve
 */
export async function resolve(specifier, context, nextResolve) {
  if (specifier.endsWith('.js') && !specifier.includes('node_modules')) {
    let candidate;
    if (specifier.startsWith('file:')) {
      try {
        candidate = fileURLToPath(specifier).replace(/\.js$/, '.ts');
      } catch {
        candidate = undefined;
      }
    } else if (specifier.startsWith('.')) {
      const parent = context.parentURL ? fileURLToPath(context.parentURL) : process.cwd();
      candidate = join(dirname(parent), specifier.replace(/\.js$/, '.ts'));
    }
    if (candidate && existsSync(candidate)) {
      return nextResolve(pathToFileURL(candidate).href, context);
    }
  }
  return nextResolve(specifier, context);
}
