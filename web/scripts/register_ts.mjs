/**
 * Register the .js → .ts resolve hook for Node source runs (tests / benches).
 */
import { register } from 'node:module';
import { pathToFileURL } from 'node:url';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
register(pathToFileURL(join(here, 'resolve_ts_hook.mjs')).href);
