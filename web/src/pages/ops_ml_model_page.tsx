import { OpsMlModel } from '@/domains/ops/ops_ml_model';
import { useOpsMlModelPageWorkspace } from '@/domains/ops/use_ops_ml_model_page_workspace';

export function OpsMlModelPage() {
  return <OpsMlModel {...useOpsMlModelPageWorkspace()} />;
}
