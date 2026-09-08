import { cn } from '@/lib/utils';

/** Sidebar product mark: layered shards + ascending event stream. */
export function AdminMark({ className }: { className?: string }) {
  return (
    <svg
      aria-hidden
      className={cn('block', className)}
      fill="none"
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M3 17.5 9.5 6.5 16 17.5H3Z"
        fill="currentColor"
        fillOpacity="0.28"
      />
      <path
        d="M6 19 12 8.5 18 19H6Z"
        fill="currentColor"
        fillOpacity="0.55"
      />
      <path d="M9.5 19 12 13.5 14.5 19" fill="currentColor" />
      <path
        d="M7 15.5 10.5 12 13 13.5 17 9"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="1.75"
      />
      <circle cx="7" cy="15.5" fill="currentColor" r="1.1" />
      <circle cx="10.5" cy="12" fill="currentColor" fillOpacity="0.85" r="1.1" />
      <circle cx="13" cy="13.5" fill="currentColor" fillOpacity="0.7" r="1.1" />
      <circle cx="17" cy="9" fill="currentColor" r="1.25" />
    </svg>
  );
}
