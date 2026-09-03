import type { DocsGuide, DocsTopic } from '@/lib/docs_types';
import { cn } from '@/lib/utils';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';

import type { DocsGuideBlock } from '@/lib/docs_types';

function DocsGuideBlockView({ block }: { block: DocsGuideBlock }) {
  switch (block.type) {
    case 'paragraph':
      return <p className="text-sm leading-relaxed text-foreground">{block.text}</p>;
    case 'heading':
      return <h3 className="text-sm font-semibold text-foreground">{block.text}</h3>;
    case 'list':
      return (
        <ul className="list-disc space-y-1.5 pl-5 text-sm leading-relaxed text-foreground">
          {block.items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      );
    case 'code':
      return (
        <pre className="overflow-x-auto rounded-sm border border-border bg-muted/40 p-3 text-xs leading-relaxed text-foreground">
          <code>{block.code}</code>
        </pre>
      );
    case 'table':
      return (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              {block.headers.map((header) => (
                <DirectoryTableHead key={header}>{header}</DirectoryTableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {block.rows.map((row) => (
              <TableRow key={row.join('|')}>
                {row.map((cell, index) => (
                  <TableCell
                    key={`${row[0]}-${index}`}
                    className={cn(index === 0 ? 'font-medium' : 'text-muted-foreground')}
                  >
                    {cell}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      );
    case 'note':
      return (
        <p className="rounded-sm border border-border bg-muted/30 px-3 py-2 text-sm leading-relaxed text-muted-foreground">
          {block.text}
        </p>
      );
    default:
      return null;
  }
}

export type DocsSectionContentProps = {
  guides?: DocsGuide[];
  topics?: DocsTopic[];
};

export function DocsSectionContent({ guides, topics }: DocsSectionContentProps) {
  return (
    <div className="grid gap-8">
      {guides?.map((guide) => (
        <article key={guide.id} className="grid gap-3">
          <h3 className="text-sm font-semibold text-foreground">{guide.title}</h3>
          <div className="grid gap-3">
            {guide.blocks.map((block, index) => (
              <DocsGuideBlockView key={`${guide.id}-${index}`} block={block} />
            ))}
          </div>
        </article>
      ))}

      {topics && topics.length > 0 ? (
        <section className="grid gap-3">
          <header className="grid gap-1">
            <h3 className="text-sm font-semibold text-foreground">Troubleshooting</h3>
            <p className="text-sm text-muted-foreground">
              Common symptoms and what to check when tracking does not match expectations.
            </p>
          </header>
          <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead className="w-[12rem]">Problem</DirectoryTableHead>
                <DirectoryTableHead className="w-[14rem]">What you see</DirectoryTableHead>
                <DirectoryTableHead>What to try</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {topics.map((topic) => (
                <TableRow key={topic.problem}>
                  <TableCell className="font-medium">{topic.problem}</TableCell>
                  <TableCell className="text-muted-foreground">{topic.symptom}</TableCell>
                  <TableCell>{topic.fix}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </section>
      ) : null}
    </div>
  );
}
