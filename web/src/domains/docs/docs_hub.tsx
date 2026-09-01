import { Navigate } from 'react-router-dom';

import { PageChrome } from '@/components/system/page_chrome';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { DocsNav } from '@/domains/docs/docs_nav';
import { DEFAULT_DOCS_SECTION_ID, getDocsSection } from '@/lib/docs_sections';

export type DocsHubProps = {
  sectionId: string | undefined;
};

export function DocsHub({ sectionId }: DocsHubProps) {
  const section = getDocsSection(sectionId);

  if (!section) {
    return <Navigate replace to={`/docs/${DEFAULT_DOCS_SECTION_ID}`} />;
  }

  return (
    <PageChrome title="Documentation">
      <p className="max-w-3xl text-sm text-muted-foreground">
        Operator notes for common admin issues. Deep runbooks live in docs/DEVELOPMENT.md and
        deploy/vendor/ on the server.
      </p>

      <div className="grid gap-6 lg:grid-cols-[14rem_minmax(0,1fr)] lg:items-start">
        <aside className="ui-surface p-3 lg:sticky lg:top-6">
          <p className="mb-2 px-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Sections
          </p>
          <DocsNav />
        </aside>

        <div className="grid gap-4">
          <header className="grid gap-1">
            <h2 className="text-base font-semibold">{section.title}</h2>
            <p className="text-sm text-muted-foreground">{section.summary}</p>
          </header>

          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[12rem]">Problem</TableHead>
                  <TableHead className="w-[14rem]">What you see</TableHead>
                  <TableHead>What to try</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {section.topics.map((topic) => (
                  <TableRow key={topic.problem}>
                    <TableCell className="align-top font-medium">{topic.problem}</TableCell>
                    <TableCell className="align-top text-muted-foreground">{topic.symptom}</TableCell>
                    <TableCell className="align-top">{topic.fix}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </PageChrome>
  );
}
