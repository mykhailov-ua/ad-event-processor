#!/usr/bin/env node

import { performance } from 'node:perf_hooks';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createCanvas } from 'canvas';
import { Window } from 'happy-dom';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const SRC = join(ROOT, 'src');

function makeMetricPoints(n, rangeMs, seed = 0) {
  const now = Date.now();
  const points = new Array(n);
  const step = rangeMs / Math.max(n - 1, 1);
  for (let i = 0; i < n; i++) {
    points[i] = {
      ts: now - rangeMs + i * step,
      value: 50 + Math.sin((i + seed) / 8) * 40 + (i % 7),
    };
  }
  return points;
}

function setupBenchDom() {
  const win = new Window({ url: 'http://localhost/', width: 640, height: 480 });
  const doc = win.document;
  doc.documentElement.style.setProperty('--accent', '#4d8fe8');
  doc.documentElement.style.setProperty('--text-muted', '#8b949e');
  doc.documentElement.style.setProperty('--border-color', '#30363d');
  doc.documentElement.style.setProperty('--bg-surface', '#0d1117');
  doc.documentElement.style.fontSize = '16px';

  const HTMLCanvasElement = win.HTMLCanvasElement;
  const origGetContext = HTMLCanvasElement.prototype.getContext;
  HTMLCanvasElement.prototype.getContext = function getContext(type, attrs) {
    if (type === '2d') {
      const w = Math.max(this.width || 640, 1);
      const h = Math.max(this.height || 200, 1);
      if (!this.__nodeCanvas || this.__nodeCanvas.width !== w || this.__nodeCanvas.height !== h) {
        this.__nodeCanvas = createCanvas(w, h);
      }
      const ctx = this.__nodeCanvas.getContext('2d', attrs);
      if (ctx) {
        const origStroke = ctx.stroke.bind(ctx);
        ctx.stroke = (path) => {
          if (path instanceof globalThis.Path2D) return;
          origStroke(path);
        };
      }
      return ctx;
    }
    return origGetContext.call(this, type, attrs);
  };

  globalThis.window = win;
  globalThis.document = doc;
  globalThis.devicePixelRatio = 2;
  globalThis.matchMedia = () => ({
    matches: true,
    media: '',
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  });
  win.dispatchEvent = () => true;

  globalThis.ResizeObserver = class {
    constructor(cb) {
      this._cb = cb;
    }
    observe() {
      this._cb?.([], this);
    }
    disconnect() {}
  };
  globalThis.HTMLElement = win.HTMLElement;
  globalThis.HTMLDivElement = win.HTMLDivElement;
  globalThis.HTMLCanvasElement = win.HTMLCanvasElement;
  globalThis.Node = win.Node;
  globalThis.Path2D = class Path2D {
    lineTo() {}
    moveTo() {}
    closePath() {}
    bezierCurveTo() {}
    quadraticCurveTo() {}
    arc() {}
    rect() {}
  };

  globalThis.requestAnimationFrame = (cb) => {
    cb(performance.now());
    return 1;
  };
  globalThis.cancelAnimationFrame = () => {};
  globalThis.getComputedStyle = (el) => win.getComputedStyle(el);

  return { win, doc };
}

function sizeElement(el, width, height) {
  el.style.width = `${width}px`;
  el.style.height = `${height}px`;
  Object.defineProperty(el, 'clientWidth', { configurable: true, value: width });
  Object.defineProperty(el, 'clientHeight', { configurable: true, value: height });
}

function bench(name, fn, opts = {}) {
  const iterations = opts.iterations ?? 200;
  const warmup = opts.warmup ?? 20;
  for (let i = 0; i < warmup; i++) fn();
  if (global.gc) global.gc();
  const heapBefore = process.memoryUsage().heapUsed;
  const t0 = performance.now();
  for (let i = 0; i < iterations; i++) fn();
  const t1 = performance.now();
  const heapAfter = process.memoryUsage().heapUsed;
  const ms = t1 - t0;
  return {
    name,
    iterations,
    ms: Number(ms.toFixed(3)),
    nsPerOp: Math.round((ms * 1e6) / iterations),
    heapDeltaBytes: heapAfter - heapBefore,
  };
}

function benchCanvas(mod, container, points) {
  const handle = mod.mountMetricChart(container, {
    title: 'Bench',
    points,
    rangeHours: 24,
    value: 100,
    color: '--accent',
  });
  const rangeMs = 24 * 60 * 60 * 1000;
  const updateBench = bench(
    'canvas metric update n=200',
    () => {
      handle.update({
        points: makeMetricPoints(200, rangeMs, Math.random() * 1000),
        value: 100,
        rangeHours: 24,
      });
    },
    { iterations: 150, warmup: 10 }
  );
  handle.destroy();
  return updateBench;
}

function benchUplot(mod, container, points) {
  const handle = mod.mountMetricChart(container, {
    title: 'Bench',
    points,
    rangeHours: 24,
    value: 100,
    color: '--accent',
  });
  const rangeMs = 24 * 60 * 60 * 1000;
  const updateBench = bench(
    'uplot metric update n=200',
    () => {
      handle.update({
        points: makeMetricPoints(200, rangeMs, Math.random() * 1000),
        value: 100,
        rangeHours: 24,
      });
    },
    { iterations: 150, warmup: 10 }
  );
  handle.destroy();
  return updateBench;
}

setupBenchDom();
const { doc } = { doc: globalThis.document };

const canvasMod = await import(
  pathToFileURL(join(ROOT, 'scripts/bench/metric_chart_canvas.js')).href
);
const uplotMod = await import(pathToFileURL(join(SRC, 'charts/metric_chart_uplot.js')).href);

const rangeMs = 24 * 60 * 60 * 1000;
const points = makeMetricPoints(200, rangeMs);

const canvasContainer = doc.createElement('div');
sizeElement(canvasContainer, 420, 220);
doc.body.appendChild(canvasContainer);

const uplotContainer = doc.createElement('div');
sizeElement(uplotContainer, 420, 220);
doc.body.appendChild(uplotContainer);

const canvasResult = benchCanvas(canvasMod, canvasContainer, points);
const uplotResult = benchUplot(uplotMod, uplotContainer, points);

await new Promise((r) => setTimeout(r, 0));

const regressionPct = (uplotResult.nsPerOp / canvasResult.nsPerOp - 1) * 100;
const verdict = regressionPct <= 30 ? 'OK (within 30% budget)' : 'REGRESSION (>30%)';

console.log('Metric chart benchmark (node --expose-gc, synthetic DOM + node-canvas 2d)');
console.log('');
console.log('| Implementation | ns/op | heap delta (bytes) |');
console.log('|----------------|------:|---------------:|');
console.log(`| canvas (baseline) | ${canvasResult.nsPerOp} | ${canvasResult.heapDeltaBytes} |`);
console.log(`| uplot | ${uplotResult.nsPerOp} | ${uplotResult.heapDeltaBytes} |`);
console.log('');
console.log(
  `Regression vs canvas: ${regressionPct >= 0 ? '+' : ''}${regressionPct.toFixed(1)}% -> ${verdict}`
);
process.exit(regressionPct <= 30 ? 0 : 1);
