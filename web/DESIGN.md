# Admin design system

Visual reference: Admin Control Panel UI kit (buttons, inputs, badges, tables, sidebar, pagination, feedback).

Implementation map:

| Kit section | Tokens | Primitives | Shell |
| :--- | :--- | :--- | :--- |
| Buttons | `admin_chrome.ts` | `components/ui/button.tsx` | — |
| Form inputs | `admin_kit.ts`, `admin_chrome.ts` | `input.tsx`, `textarea.tsx`, `select.tsx` | `search_input.tsx` |
| Checkboxes / toggles | `admin_kit.ts` | `checkbox.tsx`, `switch.tsx` | — |
| Status badges | `admin_kit.ts` | `badge.tsx` | `status_badge.tsx` |
| Tabs | — | `tabs.tsx` (`segmented`, `pill`, `underline`) | `filter_chip_group.tsx` |
| Tables | `app.css` `.admin-table--*` | `table.tsx`, `directory_table.tsx` | — |
| Pagination | — | — | `pagination_pages.tsx`, `directory_pagination_footer.tsx` |
| Sidebar | `app.css` `.admin-sidebar-*` | — | `app_sidebar.tsx` |
| Feedback | `admin_kit.ts` | `sonner.tsx` | `admin_alert.tsx` |
| Empty state | `app.css` `.admin-empty-state` | — | `empty_state.tsx` |
| Metric cards | `app.css` `.admin-metric-card` | `card.tsx` | `metric_card.tsx` |
| Progress | `app.css` `.admin-progress` | — | `progress_bar.tsx` |

## Colors

| Role | Light |
| :--- | :--- |
| Primary action | `emerald-500` / `emerald-600` hover |
| Danger | `red-500` / `red-600` |
| Focus ring | `blue-500` |
| Canvas | `slate-50` page, `white` panels |
| Sidebar | `slate-900` |
| Table header | `slate-50` bg, `slate-500` uppercase labels |
| Selected row | `blue-50` |

## Typography

| Use | Class |
| :--- | :--- |
| Body | `text-[13px] leading-[18px]` |
| Filter / table header label | `text-[11px] uppercase font-semibold text-slate-500` |
| Page title | `text-lg font-bold` |
| Numeric columns | `tabular-nums` (`admin_typography.ts`) |

## Radii

| Surface | Radius |
| :--- | :--- |
| Controls (button, input) | `5px` (`rounded-[5px]`) |
| Cards / table shell | `10px` |
| Status badge | `rounded-full` |

## Verification

```bash
cd web && npm run typecheck
bash scripts/ci/admin/web.sh
```
