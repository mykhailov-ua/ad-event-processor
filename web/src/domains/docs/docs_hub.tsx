import { Navigate } from 'react-router-dom';

import { DocsNav } from '@/domains/docs/docs_nav';
import { DocsTroubleshootingTable } from '@/domains/docs/docs_section_content';
import { PageChrome } from '@/shell/page_chrome';
import { PanelSection } from '@/shell/stat_panel';
import { Badge } from '@/components/ui/badge';
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
    <PageChrome
      title="Documentation"
      workspaceClassName="min-h-0 flex-1 border-0 bg-transparent p-0"
    >
      <div className="grid min-w-0 gap-4">
        <p className="rounded-2xl border border-border/40 bg-muted/25 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
          Operator notes for common admin issues. Deep runbooks live in{' '}
          <span className="font-mono text-xs text-foreground">docs/DEVELOPMENT.md</span> and{' '}
          <span className="font-mono text-xs text-foreground">deploy/vendor/</span> on the server.
        </p>

        <div className="grid min-w-0 gap-4 lg:grid-cols-[minmax(11rem,14rem)_minmax(0,1fr)] lg:items-start">
          <PanelSection className="h-fit lg:sticky lg:top-0" title="Sections">
            <div className="p-2">
              <DocsNav />
            </div>
          </PanelSection>

          <PanelSection
            meta={<Badge variant="outline">{section.topics.length} topics</Badge>}
            title={section.title}
          >
            <div className="grid min-w-0 gap-4 p-5">
              <p className="text-sm leading-relaxed text-muted-foreground">{section.summary}</p>
              <DocsTroubleshootingTable topics={section.topics} />
            </div>
          </PanelSection>
        </div>
      </div>
    </PageChrome>
  );
}
