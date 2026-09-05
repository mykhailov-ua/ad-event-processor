export function openCampaignCreateDialog(
  setCreateOpen: (open: boolean) => void,
  setWizardOpen: (open: boolean) => void,
): void {
  setWizardOpen(false);
  setCreateOpen(true);
}

export function openCampaignWizardSheet(
  setCreateOpen: (open: boolean) => void,
  setWizardOpen: (open: boolean) => void,
): void {
  setCreateOpen(false);
  setWizardOpen(true);
}

export function setCampaignCreateDialogOpen(
  open: boolean,
  setCreateOpen: (open: boolean) => void,
  setWizardOpen: (open: boolean) => void,
): void {
  if (open) {
    setWizardOpen(false);
  }
  setCreateOpen(open);
}

export function setCampaignWizardSheetOpen(
  open: boolean,
  setCreateOpen: (open: boolean) => void,
  setWizardOpen: (open: boolean) => void,
): void {
  if (open) {
    setCreateOpen(false);
  }
  setWizardOpen(open);
}
