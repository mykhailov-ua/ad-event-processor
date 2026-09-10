#!/usr/bin/env python3
"""Render English public offer (PART A) from deploy/vendor/PUBLIC_OFFER.md into offer.html body."""
from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SRC = ROOT / "deploy/vendor/PUBLIC_OFFER.md"
META = ROOT / "deploy/vendor/offer_meta.json"
SITE_CONFIG = ROOT / "deploy/marketing/site.config.json"
OFFER_HTML = ROOT / "deploy/marketing/offer.html"
MARKER_START = "<!-- offer-body:start -->"
MARKER_END = "<!-- offer-body:end -->"


def load_offer_version() -> str:
    env = os.environ.get("VENDOR_OFFER_VERSION", "").strip()
    if env:
        return env
    if META.is_file():
        data = json.loads(META.read_text(encoding="utf-8"))
        version = str(data.get("version", "")).strip()
        if version:
            return version
    return "2026-09-09"


def escape_html(text: str) -> str:
    return (
        text.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
    )


def inline_format(text: str) -> str:
    out = escape_html(text)
    out = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", out)
    out = re.sub(r"\*(.+?)\*", r"<em>\1</em>", out)
    out = re.sub(r"`([^`]+)`", r"<code>\1</code>", out)
    return out


def extract_english_markdown(raw: str) -> str:
    header_end = raw.find("\n---\n\n# PART A")
    if header_end == -1:
        raise ValueError("PUBLIC_OFFER.md: header or PART A marker not found")
    header = raw[:header_end].strip()
    part_a_start = raw.find("# PART A — ENGLISH VERSION")
    part_b_start = raw.find("\n# PART B —")
    if part_a_start == -1 or part_b_start == -1:
        raise ValueError("PUBLIC_OFFER.md: PART A or PART B boundary not found")
    part_a = raw[part_a_start:part_b_start].strip()
    # Drop the PART A title line; keep document header above it.
    part_a = re.sub(r"^# PART A — ENGLISH VERSION\s*\n", "", part_a, count=1)
    return header + "\n\n" + part_a


def env_or(name: str, fallback: str = "") -> str:
    return os.environ.get(name, fallback).strip()


def substitute_placeholders(text: str, offer_version: str) -> str:
    contact_email = env_or("VENDOR_CONTACT_EMAIL", "support@getbidshard.com")
    replacements = {
        "[EFFECTIVE_DATE]": offer_version,
        "[LAST_UPDATED]": offer_version,
        "[VENDOR_CONTACT_EMAIL]": contact_email,
        "[VENDOR_WEBSITE]": env_or("VENDOR_WEBSITE", "https://bidshard.com"),
        "[SUPPORT_EMAIL]": env_or("VENDOR_SUPPORT_EMAIL", contact_email),
        "[SUPPORT_TELEGRAM]": env_or("VENDOR_SUPPORT_TELEGRAM", "@bidshardsupportbot"),
    }
    for key, value in replacements.items():
        text = text.replace(key, value)
    return text


def sync_site_config_version(offer_version: str) -> None:
    if not SITE_CONFIG.is_file():
        return
    data = json.loads(SITE_CONFIG.read_text(encoding="utf-8"))
    offer = data.get("offer")
    if not isinstance(offer, dict):
        return
    if offer.get("version") == offer_version:
        return
    offer["version"] = offer_version
    SITE_CONFIG.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    print(f"render_marketing_offer: synced site.config.json offer.version={offer_version}")


def render_markdown(md: str) -> str:
    lines = md.splitlines()
    html: list[str] = []
    i = 0
    in_code = False
    code_lang = ""
    code_lines: list[str] = []
    in_ul = False
    in_ol = False
    in_table = False
    table_head_done = False

    def close_lists() -> None:
        nonlocal in_ul, in_ol
        if in_ul:
            html.append("</ul>")
            in_ul = False
        if in_ol:
            html.append("</ol>")
            in_ol = False

    def close_table() -> None:
        nonlocal in_table, table_head_done
        if in_table:
            html.append("</tbody></table>")
            in_table = False
            table_head_done = False

    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        if in_code:
            if stripped.startswith("```"):
                html.append("<pre><code>" + escape_html("\n".join(code_lines)) + "</code></pre>")
                code_lines = []
                in_code = False
            else:
                code_lines.append(line)
            i += 1
            continue

        if stripped.startswith("```"):
            close_lists()
            close_table()
            in_code = True
            code_lang = stripped[3:].strip()
            i += 1
            continue

        if not stripped:
            close_lists()
            close_table()
            i += 1
            continue

        if stripped == "---":
            close_lists()
            close_table()
            html.append("<hr />")
            i += 1
            continue

        if stripped.startswith("# "):
            close_lists()
            close_table()
            title = inline_format(stripped[2:].strip())
            if title.startswith("Public Offer Agreement /"):
                title = "Public Offer Agreement"
            html.append(f"<h1>{title}</h1>")
            i += 1
            continue

        if stripped.startswith("## "):
            close_lists()
            close_table()
            html.append(f"<h2>{inline_format(stripped[3:].strip())}</h2>")
            i += 1
            continue

        if stripped.startswith("### "):
            close_lists()
            close_table()
            html.append(f"<h3>{inline_format(stripped[4:].strip())}</h3>")
            i += 1
            continue

        if stripped.startswith("> "):
            close_lists()
            close_table()
            quote_lines = []
            while i < len(lines) and lines[i].strip().startswith(">"):
                q = lines[i].strip()
                if q == ">":
                    quote_lines.append("")
                else:
                    quote_lines.append(q[2:].strip())
                i += 1
            body = "<br />\n".join(inline_format(q) for q in quote_lines if q != "")
            html.append(f'<div class="offer-callout">{body}</div>')
            continue

        if "|" in stripped and stripped.startswith("|"):
            close_lists()
            if not in_table:
                html.append('<table class="offer-table">')
                in_table = True
                table_head_done = False
            cells = [c.strip() for c in stripped.strip("|").split("|")]
            if all(re.fullmatch(r":?-+:?", c.replace(" ", "")) for c in cells):
                i += 1
                continue
            tag = "th" if not table_head_done else "td"
            row = "".join(f"<{tag}>{inline_format(c)}</{tag}>" for c in cells)
            html.append(f"<tr>{row}</tr>")
            if not table_head_done:
                html.append("<tbody>")
                table_head_done = True
            i += 1
            continue

        if re.match(r"^\d+\.\d+(\.\d+)?\.", stripped):
            close_lists()
            close_table()
            html.append(f"<p class=\"offer-clause\">{inline_format(stripped)}</p>")
            i += 1
            continue

        if stripped.startswith("- "):
            close_table()
            if in_ol:
                html.append("</ol>")
                in_ol = False
            if not in_ul:
                html.append("<ul>")
                in_ul = True
            html.append(f"<li>{inline_format(stripped[2:].strip())}</li>")
            i += 1
            continue

        close_lists()
        close_table()
        html.append(f"<p>{inline_format(stripped)}</p>")
        i += 1

    close_lists()
    close_table()
    return "\n".join(html)


def patch_offer_html(body_html: str) -> None:
    template = OFFER_HTML.read_text(encoding="utf-8")
    if MARKER_START not in template or MARKER_END not in template:
        raise ValueError("offer.html: offer-body markers missing")
    start = template.index(MARKER_START) + len(MARKER_START)
    end = template.index(MARKER_END)
    new_html = template[:start] + "\n" + body_html + "\n    " + template[end:]
    OFFER_HTML.write_text(new_html, encoding="utf-8")


def main() -> int:
    if not SRC.is_file():
        print(f"render_marketing_offer: missing {SRC}", file=sys.stderr)
        return 1
    offer_version = load_offer_version()
    sync_site_config_version(offer_version)
    raw = SRC.read_text(encoding="utf-8")
    md = substitute_placeholders(extract_english_markdown(raw), offer_version)
    body = render_markdown(md)
    body = (
        '    <a href="index.html" class="offer-back">Back to home</a>\n\n'
        + "\n".join("    " + line if line else "" for line in body.splitlines())
        + '\n\n    <div class="offer-actions">\n'
        '      <a class="primary" data-site-cta="telegram" href="#">Request pilot via Telegram</a>\n'
        '      <a class="ghost" href="index.html">Back to home</a>\n'
        "    </div>"
    )
    patch_offer_html(body)
    print(f"render_marketing_offer: wrote English offer body into {OFFER_HTML}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
