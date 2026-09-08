import { FraudModeratorCorpus } from '@/domains/fraud/fraud_moderator_corpus';
import { useFraudModeratorCorpusPageWorkspace } from '@/domains/fraud/use_fraud_moderator_corpus_page_workspace';

export function FraudModeratorCorpusPage() {
  return <FraudModeratorCorpus {...useFraudModeratorCorpusPageWorkspace()} />;
}
