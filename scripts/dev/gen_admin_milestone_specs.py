#!/usr/bin/env python3
"""Generate deploy/vendor/ADMIN_*_MILESTONE*.md from catalog data (CUSTOMERS depth)."""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from admin_milestone_catalog import MilestoneSpec  # noqa: E402
from admin_milestone_catalog_data import ALL_SPECS  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
OUT = ROOT / "deploy" / "vendor"

SKIP_FILES = {"ADMIN_DIRECTORY_MILESTONE_CUSTOMERS.md"}

DOMAIN_RULES = (
    "`ui.mdc`, `ui-backlog.mdc`, `frontend-modular.mdc`, `react.mdc`, `anti-slop.mdc`"
)
GLOBAL_G = "G1–G10 (`ADMIN_MILESTONES_REQUIREMENTS.md`)"


def w(lines: list[str]) -> str:
    return "\n".join(lines) + "\n"


def _table(headers: list[str], rows: list[tuple[str, ...]]) -> str:
    if not rows:
        return ""

    def esc(cell: str) -> str:
        return cell.replace("|", "\\|")

    head = "| " + " | ".join(headers) + " |"
    sep = "| " + " | ".join(":---" for _ in headers) + " |"
    body = "\n".join("| " + " | ".join(esc(str(c)) for c in cells) + " |" for cells in rows)
    return w([head, sep, body, ""])


def _header(spec: MilestoneSpec) -> str:
    lines = [f"# {spec.title}", ""]
    if spec.summary:
        lines.append(spec.summary)
        lines.append("")
    if spec.gap_route:
        lines.extend(
            [
                f"**Route gap:** register `{spec.route}` in `app_routes.tsx` before `live: true`.",
                "",
            ]
        )
    lines.extend(
        [
            "**Status:** DRAFT  ",
            f"**Slug:** `{spec.slug}`  ",
            f"**Depends on:** {spec.depends}  ",
            f"**Blocks:** {spec.blocks}  ",
            f"**Pattern:** {spec.pattern}  ",
            f"**Domain rules:** {DOMAIN_RULES}",
            "",
            "---",
            "",
        ]
    )
    return w(lines)


def _section_1(spec: MilestoneSpec) -> str:
    slop_common = [
        (
            "Client filter/sort on items[]",
            "useMemo over full list",
            "URL query params + refetch only",
        ),
        (
            "Copy legacy components/",
            "Reuse FilterToolbar, table_sort",
            f"New ui/<domain>/ per {spec.pattern}",
        ),
        (
            "Silent empty on error",
            "catch → empty table",
            "ErrorBlock on blocking fetch",
        ),
        (
            "Flex page layout",
            "flex on page root",
            "CSS Grid sections per ui.mdc",
        ),
    ]
    laziness_common = [
        ("Patch legacy page in place", "Smallest diff", "Replace compose; ui/<domain>/"),
        ("`-short` only", "Fast green", "`admin_web.sh` + section 7 pasted"),
        ("Skip API gap doc", "Ship broken filter", "Section 2 API gaps table"),
    ]
    risk_rows = list(spec.risks)
    if not any(r[0] == "OpenAPI drift" for r in risk_rows):
        risk_rows.append(
            ("OpenAPI drift", "Invented TS fields", "`make openapi-types`; fields in openapi.d.ts")
        )
    if not any("PageChrome" in r[0] for r in risk_rows):
        risk_rows.append(
            (
                "PageChrome missing",
                "Foundation not shipped",
                "`test -f web/src/ui/system/page_chrome.tsx`",
            )
        )
    slop_rows = list(spec.slop) + [s for s in slop_common if s not in spec.slop][: max(0, 5 - len(spec.slop))]
    laziness_rows = list(spec.laziness) + laziness_common
    forbidden = list(spec.forbidden) or [
        '"Done" without section 7 commands and exit codes',
        '"Wired" without handler path for primary API',
        f"Client-side business rules on cold path ({GLOBAL_G})",
    ]
    return w(
        [
            "## 1. AI honesty, slop, and laziness (mandatory)",
            "",
            "### 1.1 Known hallucination risks (possible lies)",
            "",
            *_table(
                ["Risk", "Why agents lie", "How to falsify"],
                [(a, b, c) for a, b, c in risk_rows],
            ).splitlines(),
            "### 1.2 Possible AI slop (this milestone)",
            "",
            *_table(
                ["Slop pattern", "What it looks like", "What we require instead"],
                [(a, b, c) for a, b, c in slop_rows],
            ).splitlines(),
            "### 1.3 AI laziness (shortcuts to refuse)",
            "",
            *_table(
                ["Shortcut", "Agent motive", "Refuse by"],
                [(a, b, c) for a, b, c in laziness_rows],
            ).splitlines(),
            "### 1.4 Forbidden claims until verified",
            "",
            *[f"- {line}" for line in forbidden],
            "",
            "### 1.5 Doc-only delivery",
            "",
            "This file is the spec. Implementation starts when status is **REVIEW** and operator requests code.",
            "",
        ]
    )


def _section_2(spec: MilestoneSpec) -> str:
    lines = ["## 2. Scope", ""]
    if spec.in_scope:
        lines.extend(["### In scope", ""] + [f"- {item}" for item in spec.in_scope] + [""])
    if spec.out_scope:
        lines.extend(["### Out of scope", ""] + [f"- {item}" for item in spec.out_scope] + [""])
    if spec.not_on_page:
        lines.extend(
            [
                "**Not on page (explicit):** " + ", ".join(f"`{x}`" for x in spec.not_on_page),
                "",
            ]
        )
    if spec.api_gaps:
        lines.extend(
            [
                "### API gaps (block or stub)",
                "",
                *_table(
                    ["Gap", "Current state", "UI behavior until fixed"],
                    spec.api_gaps,
                ).splitlines(),
            ]
        )
    if spec.stop_triggers:
        lines.extend(
            [
                "### Stop triggers (revert slice; do not compensate)",
                "",
                *[f"- {item}" for item in spec.stop_triggers],
                "",
            ]
        )
    if not spec.in_scope and not spec.out_scope and spec.summary:
        lines.extend([spec.summary, ""])
    return w(lines)


def _section_3(spec: MilestoneSpec) -> str:
    lines = ["## 3. Contract and inputs", ""]
    if spec.contract_rows:
        lines.extend(
            _table(["Input / contract", "Source", "This milestone uses"], spec.contract_rows).splitlines()
        )
    else:
        rows: list[tuple[str, str, str]] = []
        if spec.apis:
            rows.append(("Primary API", spec.apis, "list/detail/report fetch"))
        if spec.operation_id:
            rows.append(("Operation", spec.operation_id, "OpenAPI operation_id"))
        if spec.schema_ref:
            rows.append(("Schema", spec.schema_ref, "Generated TS types"))
        if spec.handler:
            rows.append(("Handler", spec.handler, "Go handler path"))
        if spec.domain:
            rows.append(("Domain folder", f"`web/src/ui/{spec.domain}/`", "UI sections"))
            rows.append(("Helper", f"`web/src/helpers/{spec.domain}_api.ts`", "Typed fetch"))
        if spec.permission:
            rows.append(("RBAC", spec.permission, "Nav/route gate"))
        rows.append(("Global UI rules", GLOBAL_G, "All apply"))
        lines.extend(_table(["Input / contract", "Source", "This milestone uses"], rows).splitlines())
    return w(lines)


def _section_4(spec: MilestoneSpec) -> str:
    lines = ["## 4. Design spec (concrete, not intent)", ""]
    lines.append("### 4.1 Page inventory (what is on the page)")
    lines.append("")
    if spec.regions:
        lines.extend(
            _table(
                ["Region ID", "Component / section", "Purpose", "Data source"],
                [(r.id, r.component, r.purpose, r.source) for r in spec.regions],
            ).splitlines()
        )
    else:
        lines.extend(
            _table(
                ["Region", "Requirement"],
                [("PageChrome", "Title + optional freshness"), ("Content", "Primary body")],
            ).splitlines()
        )
    if spec.tabs:
        lines.extend(
            [
                "",
                "**Tabs**",
                "",
                *_table(
                    ["Tab ID", "Label", "API"],
                    spec.tabs,
                ).splitlines(),
            ]
        )
    if spec.not_on_page:
        lines.extend(
            [
                "",
                f"**Not on page (explicit):** {', '.join(spec.not_on_page)}.",
                "",
            ]
        )

    lines.extend(["### 4.2 Route and navigation (where the page lives)", ""])
    nav_rows: list[tuple[str, str]] = []
    if spec.route:
        nav_rows.append(("Path", f"`{spec.route}`"))
    if spec.nav_group:
        nav_rows.append(("Nav group", spec.nav_group))
    if spec.icon:
        nav_rows.append(("Icon", f"`{spec.icon}`"))
    if spec.permission:
        nav_rows.append(("Permission", spec.permission))
    if spec.live:
        nav_rows.append(("`live`", spec.live))
    if spec.handler:
        nav_rows.append(("Handler", spec.handler))
    if not nav_rows and spec.route:
        nav_rows = [("Path", f"`{spec.route}`")]
    lines.extend(_table(["Field", "Spec"], nav_rows).splitlines())

    lines.extend(["### 4.3 Layout and placement (grid contract)", ""])
    if spec.grid_ascii:
        lines.append("Section stack (CSS Grid on page root; no flex on page/section):")
        lines.append("")
        lines.append(spec.grid_ascii.strip())
        lines.append("")
    elif spec.grid_cols:
        lines.append(f"Column template: `--{spec.domain}-cols`: `{spec.grid_cols}`")
        lines.append("")
    if spec.grid_invariants:
        lines.extend(
            _table(["Invariant", "Value"], spec.grid_invariants).splitlines()
        )
    if spec.sortable_cols or spec.nonsortable_cols:
        lines.append("")
        if spec.sortable_cols:
            lines.append(f"**Sortable headers:** {', '.join(spec.sortable_cols)}")
        if spec.nonsortable_cols:
            lines.append(f"**Non-sortable headers:** {', '.join(spec.nonsortable_cols)}")
        lines.append("")

    lines.extend(["### 4.4 Styles and tokens (how it looks)", ""])
    style_rows = [
        ("CSS ownership", f"`web/src/ui/{spec.domain or '<domain>'}/*.module.css` only"),
        ("Tokens", "`var(--space-*)`, `var(--text-*)`, `var(--color-*)` from `tokens.css`"),
        ("Page compose", "Page file imports domain components; no CSS modules on page"),
        ("Grid class", "`.directory` or domain root; `.grid` on content when grid page"),
    ]
    if spec.grid_cols:
        style_rows.append(("Grid token", f"`--{spec.domain}-cols`: {spec.grid_cols}"))
    lines.extend(_table(["Topic", "Spec"], style_rows).splitlines())

    lines.extend(["### 4.5 API and cold path (what the browser does not compute)", ""])
    if spec.url_params:
        lines.extend(
            [
                "**URL param mapping:**",
                "",
                *_table(
                    ["URL param", "API query", "Default", "Notes"],
                    [(p.name, p.api, p.default, p.notes) for p in spec.url_params],
                ).splitlines(),
            ]
        )
    cold_rows: list[tuple[str, str, str]] = []
    if spec.pattern == "admin_directory_pattern" or spec.url_params:
        cold_rows.extend(
            [
                ("Pagination", "`limit`, `offset`, `total`", "URL → refetch on page change"),
                ("Sort", "OpenAPI `sort`/`order` when present", "URL → refetch on Apply / header click"),
            ]
        )
    if spec.tabs:
        cold_rows.append(("Tabs", "Separate GET per tab", "No client merge of tab payloads"))
    if spec.pattern == "admin_detail_pattern" or spec.tabs:
        cold_rows.extend(
            [
                ("GET detail", "Full DTO from handler", "Render as returned"),
                ("PATCH", "Fields ⊆ OpenAPI Patch*Request", "apiConfirmed after 2xx"),
            ]
        )
    if spec.pattern == "admin_report_pattern":
        cold_rows.extend(
            [
                ("Report rows", "Handler row keys", "No client aggregation"),
                ("Export", "POST `/api/v1/reports/jobs` + poll", "When API supports"),
            ]
        )
    if not cold_rows:
        cold_rows = [
            ("Primary fetch", spec.apis or "OpenAPI", "Server owns business rules"),
            ("Display", "`*_display`, `status_label`", "Render only"),
        ]
    lines.extend(_table(["Concern", "Handler / OpenAPI", "Browser"], cold_rows).splitlines())
    if spec.fetch_example:
        lines.extend(["", "Fetch example:", "", "```", spec.fetch_example.strip(), "```", ""])

    lines.extend(["### 4.6 File map (where code lands)", ""])
    if spec.files:
        lines.extend(
            _table(["Path", "Role"], spec.files).splitlines()
        )
    elif spec.domain:
        lines.extend(
            _table(
                ["Path", "Role"],
                [
                    (f"web/src/pages/{spec.legacy or spec.domain + '_page.tsx'}", "Compose"),
                    (f"web/src/ui/{spec.domain}/*", "Sections + CSS modules"),
                    (f"web/src/helpers/{spec.domain}_api.ts", "API helpers"),
                ],
            ).splitlines()
        )
    if spec.legacy_remove:
        lines.extend(
            [
                "",
                f"**Remove from this route (legacy):** {', '.join(spec.legacy_remove)}.",
                "",
            ]
        )
    if spec.legacy and spec.legacy not in ("gap", "—"):
        lines.extend([f"**Legacy page:** `{spec.legacy}`", ""])

    lines.extend(["### 4.7 Pitfalls checklist (avoid documented failures)", ""])
    base_pitfalls = [
        ("Client filter/sort on `items[]`", "Forbidden; only URL params in 4.5"),
        ("Portal filter listbox", "Inline drop; wrapper width 100%"),
        ("Double freshness chrome", "FreshnessBadge in PageChrome slot only"),
        ("Silent `catch` → empty table", "ErrorBlock on list error"),
        ("Piecemeal edit", "Atomic PR: all paths in 4.6 + page replace"),
    ]
    pitfall_rows = list(spec.pitfalls) + [p for p in base_pitfalls if p not in spec.pitfalls]
    lines.extend(_table(["Pitfall", "Prevention in this milestone"], pitfall_rows).splitlines())
    return w(lines)


def _section_5(spec: MilestoneSpec) -> str:
    lines = ["## 5. Implementation plan (ordered)", ""]
    if spec.impl_steps:
        lines.extend(
            _table(["Step", "Artifact(s)", "Action", "Done when"], spec.impl_steps).splitlines()
        )
    else:
        domain = spec.domain or "<domain>"
        lines.extend(
            [
                "| Step | Artifact(s) | Action | Done when |",
                "| :--- | :--- | :--- | :--- |",
                f"| 1 | OpenAPI + handler | Confirm contract | openapi_gate or manual |",
                f"| 2 | make openapi-types | Regenerate types | typecheck |",
                f"| 3 | web/src/helpers/{domain}_api.ts | Typed helpers | compiles |",
                f"| 4 | web/src/ui/{domain}/* | Sections per 4.1–4.4 | surface gate |",
                f"| 5 | page compose + route | URL sync | route loads |",
                "",
            ]
        )
    return w(lines)


def _section_6(spec: MilestoneSpec) -> str:
    lines = ["## 6. SLA and performance", ""]
    if spec.sla_rows:
        lines.extend(
            _table(["Surface / path", "Metric", "Ceiling", "How measured"], spec.sla_rows).splitlines()
        )
    else:
        lines.extend(["N/A — admin cold path; not `/track`.", ""])
    return w(lines)


def _section_7(spec: MilestoneSpec) -> str:
    verify_cmds = spec.extra_verify or [
        "cd web && npm run typecheck",
        "bash scripts/ci/check_ui_surface_gate.sh",
        "bash scripts/ci/admin_web.sh",
    ]
    lines = [
        "## 7. Verification (paste in PR)",
        "",
        "```bash",
        *verify_cmds,
        "```",
        "",
    ]
    if spec.manual_checks:
        lines.extend(
            _table(["Check", "Command / procedure", "Pass criteria"], spec.manual_checks).splitlines()
        )
    else:
        lines.extend(
            _table(
                ["Check", "Pass criteria"],
                [
                    ("typecheck", "exit 0"),
                    ("check_ui_surface_gate.sh", "exit 0"),
                    ("admin_web.sh", "exit 0"),
                ],
            ).splitlines()
        )
    lines.extend(["", "PR body must paste commands **actually run** with exit codes.", ""])
    return w(lines)


def _section_8_9(spec: MilestoneSpec) -> str:
    domain = spec.domain or "<domain>"
    legacy_note = ""
    if spec.legacy and spec.legacy not in ("gap", "—"):
        legacy_note = f"- [ ] Legacy `{spec.legacy}` replaced or delegates to `ui/{domain}/`"
    return w(
        [
            "## 8. Definition of done",
            "",
            "- [ ] Sections 1.1–1.4, 4.1–4.7, 5, 6, 7 complete",
            f"- [ ] {GLOBAL_G} satisfied",
            legacy_note,
            "- [ ] Verification output pasted in PR",
            "",
            "## 9. Rollback",
            "",
            f"Revert `web/src/ui/{domain}/`, helpers, and page compose in one slice; no half migration.",
            "",
        ]
    )


def render_spec(spec: MilestoneSpec) -> str:
    parts = [
        _header(spec),
        _section_1(spec),
        _section_2(spec),
        _section_3(spec),
        _section_4(spec),
        _section_5(spec),
        _section_6(spec),
        _section_7(spec),
        _section_8_9(spec),
    ]
    return "".join(parts)


def main() -> None:
    written = 0
    skipped: list[str] = []
    for spec in ALL_SPECS:
        if spec.filename in SKIP_FILES:
            skipped.append(spec.filename)
            print(f"skip {spec.filename} (reference spec preserved)")
            continue
        path = OUT / spec.filename
        path.write_text(render_spec(spec), encoding="utf-8")
        written += 1
        print(f"wrote {path.relative_to(ROOT)}")
    for name in sorted(SKIP_FILES):
        if name not in skipped and (OUT / name).exists():
            print(f"skip {name} (reference spec preserved; not in ALL_SPECS)")
    print(f"written: {written}")
    if skipped:
        print(f"skipped: {', '.join(skipped)}")
    total = len(list(OUT.glob("ADMIN_*_MILESTONE*.md")))
    print(f"total ADMIN milestone files on disk: {total}")


if __name__ == "__main__":
    main()
