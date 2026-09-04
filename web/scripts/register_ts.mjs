// Registers test_resolve.mjs so bench scripts can import @/ paths without a bundler.
import { register } from 'node:module';

register('./test_resolve.mjs', import.meta.url);
