import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { buildSync } from 'esbuild';

const WEB_ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const REPO_ROOT = join(WEB_ROOT, '..');
const SRC = join(WEB_ROOT, 'src/static/track.js');
const TRACK_PIXEL_OUT = join(REPO_ROOT, 'internal/track/track_pixel.js');

function buildTrackPixelBundle() {
  const result = buildSync({
    entryPoints: [SRC],
    bundle: true,
    minify: true,
    format: 'iife',
    write: false,
    legalComments: 'none',
    target: 'es2018',
  });

  const code = result.outputFiles[0].text;
  return code;
}

function copyToDist(distRoot) {
  const distTrack = join(distRoot, 'src/static/track.js');
  mkdirSync(dirname(distTrack), { recursive: true });
  writeFileSync(distTrack, buildTrackPixelBundle());
}

export function buildTrackPixel({ check = false, distRoot } = {}) {
  if (!existsSync(SRC)) {
    throw new Error(`missing canonical track source: ${SRC}`);
  }

  const built = buildTrackPixelBundle();

  if (check) {
    if (!existsSync(TRACK_PIXEL_OUT)) {
      console.error(
        `track_pixel drift: missing ${TRACK_PIXEL_OUT}; run node web/scripts/build_track_pixel.mjs`
      );
      process.exit(1);
    }
    const existing = readFileSync(TRACK_PIXEL_OUT, 'utf8');
    if (existing !== built) {
      const existingHash = createHash('sha256').update(existing).digest('hex').slice(0, 12);
      const builtHash = createHash('sha256').update(built).digest('hex').slice(0, 12);
      console.error(
        `track_pixel drift: ${TRACK_PIXEL_OUT} does not match web/src/static/track.js (existing=${existingHash} built=${builtHash})`
      );
      console.error('Run: node web/scripts/build_track_pixel.mjs');
      process.exit(1);
    }
    console.log('track_pixel parity: OK');
    return built;
  }

  writeFileSync(TRACK_PIXEL_OUT, built);
  console.log(`track_pixel: wrote ${TRACK_PIXEL_OUT} (${built.length} bytes)`);

  if (distRoot) {
    copyToDist(distRoot);
    console.log(`track_pixel: wrote ${join(distRoot, 'src/static/track.js')}`);
  }

  return built;
}

const checkMode = process.argv.includes('--check');
const distArg = process.argv.find((arg) => arg.startsWith('--dist='));
const distRoot = distArg ? distArg.slice('--dist='.length) : undefined;

buildTrackPixel({ check: checkMode, distRoot });
