import type { LanderBlock } from '@/domains/creative/lander_block_model';

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

function renderBlock(block: LanderBlock): string {
  switch (block.type) {
    case 'hero':
      return `<section class="hero"><h1>${escapeHtml(block.title)}</h1><p>${escapeHtml(block.subtitle)}</p></section>`;
    case 'cta':
      return `<section class="cta"><a class="cta-button" href="${escapeHtml(block.href)}">${escapeHtml(block.label)}</a></section>`;
    case 'image':
      return `<section class="image"><img src="${escapeHtml(block.src)}" alt="${escapeHtml(block.alt)}" loading="lazy" /></section>`;
    case 'html':
      return `<section class="html-block">${block.content}</section>`;
    default:
      return '';
  }
}

export function compileLanderBlocksToHtml(
  blocks: LanderBlock[],
  pageTitle = 'Landing page'
): string {
  const body = blocks.map(renderBlock).join('\n');
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>${escapeHtml(pageTitle)}</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 0; padding: 24px; line-height: 1.5; }
    .hero h1 { margin: 0 0 8px; font-size: 2rem; }
    .cta-button { display: inline-block; padding: 12px 20px; background: #111; color: #fff; text-decoration: none; border-radius: 6px; }
    .image img { max-width: 100%; height: auto; }
  </style>
</head>
<body>
${body}
</body>
</html>
`;
}
