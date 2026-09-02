// Registers test_resolve.mjs so `npm test` can import @/ paths without a bundler.
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { register } from 'node:module';

register('./test_resolve.mjs', import.meta.url);
