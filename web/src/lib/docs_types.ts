export type DocsTopic = {
  problem: string;
  symptom: string;
  fix: string;
};

export type DocsGuideBlock =
  | { type: 'paragraph'; text: string }
  | { type: 'heading'; text: string }
  | { type: 'list'; items: string[] }
  | { type: 'code'; code: string }
  | { type: 'table'; headers: string[]; rows: string[][] }
  | { type: 'note'; text: string };

export type DocsGuide = {
  id: string;
  title: string;
  blocks: DocsGuideBlock[];
};

export type DocsSection = {
  id: string;
  title: string;
  summary: string;
  audience?: 'client' | 'operator';
  guides?: DocsGuide[];
  topics?: DocsTopic[];
};
