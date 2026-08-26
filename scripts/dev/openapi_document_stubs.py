#!/usr/bin/env python3
"""Add hand-maintained OpenAPI path refs and documented_routes entries for stub routes."""

from __future__ import annotations

import re
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
OPENAPI = REPO / "api/openapi/openapi.yaml"
DOCUMENTED = REPO / "internal/openapi/documented_routes.go"

NEW_PATH_FILES = {
    "paths/platform.yaml": [
        "GET /api/v1/audit",
        "GET /api/v1/audit/export",
        "POST /api/v1/consent",
        "GET /api/v1/customers",
        "GET /api/v1/customers/{id}",
        "GET /api/v1/disputes",
        "GET /api/v1/eula",
        "POST /api/v1/eula/accept",
        "GET /api/v1/license/status",
        "POST /api/v1/license/apply",
        "GET /api/v1/meta",
        "GET /api/v1/publisher/dashboard",
        "GET /api/v1/publisher/statements",
        "GET /api/v1/recon/runs",
        "GET /api/v1/report-schedules",
        "POST /api/v1/report-schedules",
        "GET /api/v1/report-schedules/{id}",
        "PUT /api/v1/report-schedules/{id}",
        "DELETE /api/v1/report-schedules/{id}",
        "GET /api/v1/settings/platform",
        "PATCH /api/v1/settings/platform",
        "POST /api/v1/settings/platform/apply",
        "POST /api/v1/settings/platform/bootstrap",
        "GET /api/v1/support/feedback/meta",
        "POST /api/v1/support/feedback",
        "GET /api/v1/team/overview",
        "POST /api/v1/team/members",
        "PATCH /api/v1/team/members/{id}",
        "GET /api/v1/team/budget-approvals",
        "POST /api/v1/team/budget-approvals/{id}/approve",
        "POST /api/v1/team/budget-approvals/{id}/deny",
    ],
    "paths/rtb.yaml": [
        "GET /api/v1/rtb/deals",
        "POST /api/v1/rtb/deals",
        "GET /api/v1/rtb/deals/{id}",
        "PATCH /api/v1/rtb/deals/{id}",
        "DELETE /api/v1/rtb/deals/{id}",
        "GET /api/v1/rtb/integration-profile",
        "GET /api/v1/rtb/shadow-diff",
        "GET /api/v1/rtb/reconcile/export",
        "POST /api/v1/rtb/floors/apply",
        "POST /api/v1/rtb/validate-bid-request",
    ],
    "paths/telegram_ops.yaml": [
        "POST /api/v1/telegram/validate",
        "POST /api/v1/telegram/clicks",
        "POST /api/v1/telegram/webhook/{bot_id}",
        "POST /api/v1/telegram/deeplink-tokens",
        "GET /api/v1/telegram/deeplink-tokens/{token}",
        "GET /api/v1/telegram/bots",
        "GET /api/v1/telegram/bots/{id}",
        "PUT /api/v1/telegram/bots/{id}",
        "GET /api/v1/telegram/postbacks",
        "POST /api/v1/telegram/postbacks",
        "PUT /api/v1/telegram/postbacks/{id}",
        "DELETE /api/v1/telegram/postbacks/{id}",
        "POST /api/v1/telegram/postbacks/{id}/test",
    ],
    "paths/campaigns.yaml": [
        "GET /api/v1/campaigns/{id}/conversion-mappings",
        "PUT /api/v1/campaigns/{id}/conversion-mappings",
        "GET /lp/{lander_id}/{path...}",
        "GET /lp-preview/{lander_id}/{path...}",
    ],
    "paths/integrations.yaml": [
        "GET /api/v1/integration/affiliate-status-presets",
    ],
    "paths/billing.yaml": [
        "POST /api/v1/billing/crypto/webhook",
    ],
}

NEW_TAGS = [
    ("audit", "Admin audit log and CSV export"),
    ("customers", "Customer directory (admin)"),
    ("disputes", "Payment dispute and chargeback rows"),
    ("eula", "End-user license acceptance"),
    ("license", "JWT license status and apply"),
    ("meta", "Product metadata and bootstrap flags"),
    ("consent", "Signed consent records"),
    ("recon", "Ledger reconciliation run history"),
    ("report-schedules", "Scheduled report delivery cron jobs"),
    ("settings", "Platform install configuration"),
    ("support", "Operator feedback to vendor"),
    ("team", "Team members and budget approvals"),
    ("publisher", "Scoped publisher dashboard and statements"),
    ("rtb", "OpenRTB deals, shadow diff, and floor optimizer"),
]


def encode_path(path: str) -> str:
    return path.replace("/", "~1")


def path_ref(rel_file: str, http_path: str) -> str:
    enc = encode_path(http_path)
    return f"  {http_path}:\n    $ref: {rel_file}#/paths/{enc}"


def main() -> None:
    routes: list[str] = []
    for keys in NEW_PATH_FILES.values():
        routes.extend(keys)
    routes = sorted(set(routes))

    text = OPENAPI.read_text()
    for rel, keys in NEW_PATH_FILES.items():
        for route in keys:
            _, http_path = route.split(" ", 1)
            block = path_ref(rel, http_path)
            if block.strip().split("\n")[0] + "\n" in text:
                continue
            if f"  {http_path}:\n" in text:
                continue
            insert_at = text.find("components:")
            if insert_at < 0:
                raise SystemExit("components: anchor missing")
            text = text[:insert_at] + block + "\n" + text[insert_at:]

    for name, desc in NEW_TAGS:
        tag_line = f"  - name: {name}\n    description: {desc}"
        if f"  - name: {name}\n" in text:
            continue
        anchor = text.find("paths:")
        text = text[:anchor] + tag_line + "\n" + text[anchor:]

    text = re.sub(
        r"version: 0\.5\.0",
        "version: 0.6.0",
        text,
        count=1,
    )
    text = re.sub(
        r"Remaining routes are stubbed in `paths/_generated_routes\.yaml`\.",
        "All `/api/v1` catalog routes are hand-documented or explicitly allowlisted.",
        text,
        count=1,
    )
    OPENAPI.write_text(text)

    doc = DOCUMENTED.read_text()
    marker = "\t\"PUT /api/v1/views/{id}\",\n}"
    additions = "".join(f'\t"{r}",\n' for r in routes if f'"{r}"' not in doc)
    if additions and marker in doc:
        doc = doc.replace(marker, additions + marker)
        DOCUMENTED.write_text(doc)
    elif additions:
        raise SystemExit("documented_routes.go marker not found")

    print(f"openapi_document_stubs: added {len(routes)} routes")


if __name__ == "__main__":
    main()
