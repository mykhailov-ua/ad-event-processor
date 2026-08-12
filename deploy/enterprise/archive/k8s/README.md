# K3s deployment archive (Enterprise / removed from appliance SKU)

These scripts supported an optional **k3s** install path. The boxed product ships **Docker Compose `single_vps`** only (`docs/QUICKSTART.md`).

**Removed from active tree (2026-08):** installer profile `k8s_k3s`, `scripts/ops/*k3s*`, and `deploy/monitoring/prometheus-k3s.yaml`. Kept here for Enterprise SOW or historical reference — not wired into `espx-install`, CI, or QUICKSTART.

| File | Former role |
| --- | --- |
| `install_k3s.sh` | Bootstrap k3s on a node |
| `import_image.sh` | Import container images into k3s containerd |
| `hot_path_up.sh` / `cold_path_up.sh` | Deploy hot/cold path manifests |
| `hot_path_smoke.sh` / `cold_path_smoke.sh` | Post-deploy smoke checks |
| `render_prometheus_k3s.sh` | Render `prometheus-k3s.yaml` with node IP |
| `prometheus-k3s.yaml` | Scrape config for k3s NodePort metrics |

Do not reference `k8s_k3s` in new installer or admin UI code.
