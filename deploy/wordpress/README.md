# WordPress integration (optional)

Package path: `deploy/wordpress/ad-event-tracker/`

## Install

1. Zip the `ad-event-tracker` directory or copy it to `wp-content/plugins/ad-event-tracker`.
2. Activate **Ad Event Tracker** in WordPress admin.
3. Open **Settings -> Ad Event Tracker** and set:
   - Control plane URL (admin API origin, e.g. `https://control.example.com`)
   - Tracker URL (e.g. `https://trk.example.com`)
   - Bearer API token with `campaigns:read` or `campaigns:write`
   - Default campaign UUID

## Shortcode

```
[aed_click_link label="Buy now" campaign_id="00000000-0000-4000-8000-000000000001"]
```

The shortcode calls `POST /api/v1/tracker/clicks` server-side and renders a link to the returned `click_url`.

## track.js

When tracker URL is configured, the plugin enqueues `{tracker_url}/static/track.js` on public pages for zero-redirect browser events per `docs/INTEGRATIONS.md`.

## Version pin

Plugin version `1.0.0` targets Click API contract `POST /api/v1/tracker/clicks` (OpenAPI `trackerMintClick`).
