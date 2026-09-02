// Node custom resolver: maps @/ imports to web/src for unit tests (see test_aliases.mjs).
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'src');

export async function resolve(specifier, context, nextResolve) {
  if (!specifier.startsWith('@/')) {
    return nextResolve(specifier, context);
  }

  const rel = specifier.slice(2);
  const candidates = [`${rel}.ts`, `${rel}.tsx`, `${rel}.js`, path.join(rel, 'index.ts')];
  if (rel.endsWith('.ts') || rel.endsWith('.tsx') || rel.endsWith('.js')) {
    candidates.unshift(rel);
  }
  for (const candidate of candidates) {
    const url = pathToFileURL(path.join(SRC, candidate)).href;
    return { shortCircuit: true, url };
  }

  return nextResolve(specifier, context);
}
