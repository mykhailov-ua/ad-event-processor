#!/usr/bin/env node
import { runJsonParseColdPathBench } from '@/lib/perf/benches/json_parse.ts';

console.log('admin web bench: cold path (JSON parse / prefs)');
runJsonParseColdPathBench();
console.log('admin web bench: cold path PASSED');
