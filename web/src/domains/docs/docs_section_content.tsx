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
        <pre className="ui-code-block overflow-x-auto text-xs leading-relaxed">
          <code>{block.code}</code>
        </pre>
      );
    case 'table':
      return (
        <DirectoryTable horizontalScroll className="min-w-0">
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
                    className={cn(
                      'min-w-[8rem] align-top break-words',
                      index === 0 ? 'font-medium' : 'text-muted-foreground',
                    )}
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
        <p className="rounded-md border border-border bg-muted/30 px-3 py-2 text-sm leading-relaxed text-muted-foreground">
          {block.text}
        </p>
      );
    default:
      return null;
  }
}

export function DocsTroubleshootingTable({ topics }: { topics: DocsTopic[] }) {
  return (
    <DirectoryTable horizontalScroll className="min-w-0">
      <TableHeader>
        <TableRow>
          <DirectoryTableHead className="min-w-[11rem]">Problem</DirectoryTableHead>
          <DirectoryTableHead className="min-w-[12rem]">What you see</DirectoryTableHead>
          <DirectoryTableHead className="min-w-[16rem]">What to try</DirectoryTableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {topics.map((topic) => (
          <TableRow key={topic.problem}>
            <TableCell className="min-w-[11rem] align-top break-words font-medium">
              {topic.problem}
            </TableCell>
            <TableCell className="min-w-[12rem] align-top break-words text-muted-foreground">
              {topic.symptom}
            </TableCell>
            <TableCell className="min-w-[16rem] align-top break-words text-sm leading-relaxed">
              {topic.fix}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DirectoryTable>
  );
}

export type DocsSectionContentProps = {
  guides?: DocsGuide[];
  topics?: DocsTopic[];
};

export function DocsSectionContent({ guides, topics }: DocsSectionContentProps) {
  return (
    <div className="grid min-w-0 gap-8">
      {guides?.map((guide) => (
        <article key={guide.id} className="grid min-w-0 gap-3">
          <h3 className="text-sm font-semibold text-foreground">{guide.title}</h3>
          <div className="grid min-w-0 gap-3">
            {guide.blocks.map((block, index) => (
              <DocsGuideBlockView key={`${guide.id}-${index}`} block={block} />
            ))}
          </div>
        </article>
      ))}

      {topics && topics.length > 0 ? (
        <section className="grid min-w-0 gap-3">
          <header className="grid gap-1">
            <h3 className="text-sm font-semibold text-foreground">Troubleshooting</h3>
            <p className="text-sm text-muted-foreground">
              Common symptoms and what to check when tracking does not match expectations.
            </p>
          </header>
          <DocsTroubleshootingTable topics={topics} />
        </section>
      ) : null}
    </div>
  );
}
