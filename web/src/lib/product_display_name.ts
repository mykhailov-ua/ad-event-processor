/** Browser tab title. Wire/env slug stays ad-event-processor. */
export const productPageTitle = 'AEP Admin (Ad Event Processor - AEP)';

/** Short chrome label (sidebar, auth copy). */
export const productDisplayName = 'AEP Admin';

export function productSignInPageTitle(): string {
  return `Sign in - ${productPageTitle}`;
}

export function productConsoleTagline(): string {
  return 'Ad Event Processor operator console';
}
