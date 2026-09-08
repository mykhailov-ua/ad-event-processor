// Node custom resolver: maps @/ and extensionless relative imports for unit tests (see test_aliases.mjs).
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'src');

function extensionCandidates(specifier) {
  if (/\.(ts|tsx|js|mjs|cjs|json)$/.test(specifier)) {
    const candidates = [specifier];
    if (specifier.endsWith('.js')) {
      const base = specifier.slice(0, -3);
      candidates.unshift(`${base}.tsx`, `${base}.ts`);
    }
    if (specifier.endsWith('.ts')) {
      candidates.push(`${specifier.slice(0, -3)}.tsx`);
    }
    return candidates;
  }
  return [
    `${specifier}.ts`,
    `${specifier}.tsx`,
    `${specifier}.js`,
    path.join(specifier, 'index.ts'),
    path.join(specifier, 'index.tsx'),
  ];
}

function firstExisting(baseDir, rel) {
  for (const candidate of extensionCandidates(rel)) {
    const full = path.resolve(baseDir, candidate);
    if (fs.existsSync(full)) {
      return pathToFileURL(full).href;
    }
  }
  return null;
}

export async function resolve(specifier, context, nextResolve) {
  if (specifier.startsWith('@/')) {
    const rel = specifier.slice(2);
    const url = firstExisting(SRC, rel);
    if (url) {
      return { shortCircuit: true, url };
    }
  }

  if ((specifier.startsWith('./') || specifier.startsWith('../')) && context.parentURL) {
    const base = path.dirname(fileURLToPath(context.parentURL));
    const url = firstExisting(base, specifier);
    if (url) {
      return { shortCircuit: true, url };
    }
  }

  return nextResolve(specifier, context);
}
