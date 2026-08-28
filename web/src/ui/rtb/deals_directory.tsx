import * as auth from '../../helpers/auth.js';
import { can } from '../../helpers/permissions.js';
import type { RtbDeal } from '../../helpers/rtb_api.js';
import { ContextBar } from '../shell/context_bar.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { Button } from '../system/button.js';
import { DealFormModal } from './deal_form_modal.js';
import { DealsGrid } from './deals_grid.js';
import styles from './deals_directory.module.css';

export type DealsDirectoryProps = {
  items: RtbDeal[];
  loading: boolean;
  error: unknown;
  modalOpen: boolean;
  modalMode: 'create' | 'edit';
  editingDeal: RtbDeal | null;
  modalBusy: boolean;
  modalError: string | null;
  onOpenCreate: () => void;
  onOpenEdit: (deal: RtbDeal) => void;
  onCloseModal: () => void;
  onSubmitModal: (body: {
    deal_id: string;
    customer_id: string;
    floor_micro?: number;
    pacing?: string;
    seats?: number;
  }) => void;
  onDelete: (deal: RtbDeal) => void;
};

export function DealsDirectory({
  items,
  loading,
  error,
  modalOpen,
  modalMode,
  editingDeal,
  modalBusy,
  modalError,
  onOpenCreate,
  onOpenEdit,
  onCloseModal,
  onSubmitModal,
  onDelete,
}: DealsDirectoryProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canWrite = can(permissions, 'rtb:write');

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load RTB deals" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome title="RTB deals" badge={<LoadingCountBadge loading={loading} label={`${items.length} deals`} />} />
      <ContextBar parentLabel="RTB" parentTo="/rtb/integration" currentLabel="Deals" />
      {canWrite ? (
        <div className={styles.toolbar}>
          <Button variant="primary" onClick={onOpenCreate}>
            Create deal
          </Button>
        </div>
      ) : null}
      <div className={styles.content}>
        <DealsGrid
          items={items}
          loading={loading}
          canWrite={canWrite}
          onEdit={onOpenEdit}
          onDelete={onDelete}
          onCreate={onOpenCreate}
        />
      </div>
      <DealFormModal
        open={modalOpen}
        mode={modalMode}
        initial={editingDeal}
        busy={modalBusy}
        error={modalError}
        onClose={onCloseModal}
        onSubmit={onSubmitModal}
      />
    </div>
  );
}
