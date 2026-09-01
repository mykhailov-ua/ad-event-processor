import { Navigate, Route, Routes, useParams } from 'react-router-dom';

import { DocsHub } from '@/domains/docs/docs_hub';
import { DEFAULT_DOCS_SECTION_ID } from '@/lib/docs_sections';

function DocsSectionPage() {
  const { sectionId } = useParams<{ sectionId: string }>();
  return <DocsHub sectionId={sectionId} />;
}

export function DocsPage() {
  return (
    <Routes>
      <Route element={<Navigate replace to={`/docs/${DEFAULT_DOCS_SECTION_ID}`} />} index />
      <Route element={<DocsSectionPage />} path=":sectionId" />
    </Routes>
  );
}
