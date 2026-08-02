export const INDEX_HTML = `<!doctype html>
<html lang="en" data-theme="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="BidShard Admin — control plane operator interface." />
    <title>BidShard Admin</title>
    __STYLES__
  </head>
  <body>
    <div id="root"></div>
    __SCRIPTS__
  </body>
</html>
`;

export const LOGIN_HTML = `<!doctype html>
<html lang="en" data-theme="dark">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="BidShard Admin — sign in" />
    <title>Sign in — BidShard Admin</title>
    __STYLES__
  </head>
  <body>
    <div id="root"></div>
    __SCRIPTS__
  </body>
</html>
`;

/**
 * @param {string} template
 * @param {string[]} scripts
 * @param {string[]} styles
 */
export function renderHtml(template, scripts, styles) {
  const styleTags = styles
    .map((href) => `<link rel="stylesheet" href="${href}" />`)
    .join('\n    ');
  const scriptTags = scripts
    .map((src) => `<script type="module" src="${src}"></script>`)
    .join('\n    ');
  return template
    .replace('__STYLES__', styleTags)
    .replace('__SCRIPTS__', scriptTags);
}

/**
 * @param {string} distPath
 */
function publicPath(distPath) {
  return '/' + distPath.replace(/^dist[/\\]/, '').replace(/\\/g, '/');
}

/**
 * @param {import('esbuild').Metafile} meta
 * @param {string} entryPoint
 */
export function assetsForEntry(meta, entryPoint) {
  const want = entryPoint.replace(/\\/g, '/');
  const entryBase = want.endsWith('main.js') ? 'main' : want.endsWith('login.js') ? 'login' : '';
  const scripts = [];
  const styles = [];

  for (const [outPath, info] of Object.entries(meta.outputs)) {
    const ep = (info.entryPoint ?? '').replace(/\\/g, '/');
    const pub = publicPath(outPath);
    if (ep === want && outPath.endsWith('.js')) {
      scripts.push(pub);
      continue;
    }
    if (!outPath.endsWith('.css') || !entryBase) continue;
    const name = outPath.split('/').pop() ?? '';
    if (name === `${entryBase}.css` || (name.startsWith(`${entryBase}-`) && name.endsWith('.css'))) {
      styles.push(pub);
    }
  }

  return { scripts, styles };
}
