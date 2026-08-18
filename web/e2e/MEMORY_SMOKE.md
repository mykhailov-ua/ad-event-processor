# Admin UI memory smoke

Manual checklist before release (target ~30 min session):

1. Open `/campaigns`, paginate 5 pages, open/close 3 campaign details.
2. Open `/reports/placements` with 90d range, load more twice.
3. Toggle theme dark/light 10 times.
4. Open `/ops`, switch tabs (summary / outbox / doctor).
5. In DevTools Performance: heap after navigation should not grow monotonically across loops.

Automated guards: `scripts/ci/admin_bundle_gate.sh` (generous uncompressed JS limits; lazy chunks allowed).
