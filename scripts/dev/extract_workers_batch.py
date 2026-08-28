#!/usr/bin/env python3
"""Extract controlplane/workers.go blocks into domain packages."""

from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WORKERS = ROOT / "internal/controlplane/workers.go"
OP_LEASE = ROOT / "internal/controlplane/operation_lease.go"
ALIASES = ROOT / "internal/controlplane/worker_wire.go"

lines = WORKERS.read_text().splitlines(keepends=True)


def chunk(start: int, end: int) -> str:
    return "".join(lines[start - 1 : end])


def write(path: Path, body: str, imports: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(f"package {path.parent.name}\n\n{imports}\n{body}")


def replace(body: str, pairs: list[tuple[str, str]]) -> str:
    for old, new in pairs:
        body = body.replace(old, new)
    return body


SERVICE_HOST_PAIRS = [
    ("*Service", "Host"),
    ("svc *Service", "host Host"),
    ("w.svc", "w.host"),
    ("j.svc", "j.host"),
    ("o.svc", "o.host"),
    ("func New", "func New"),  # noop anchor
]

# --- domain extractions (1-based inclusive line ranges) ---

# nodeadmin scorer workers
scorer_body = chunk(210, 273)
scorer_body = scorer_body.replace("*NodeCapacityScorer", "*NodeCapacityScorer")
scorer_body = scorer_body.replace("func NewNodeCapacityScorerWorker(svc *Service)", "func NewCapacityScorerWorker(host ScorerHost)")
scorer_body = scorer_body.replace("func NewGlobalRegionTrafficScorerWorker(svc *Service)", "func NewGlobalTrafficScorerWorker(host ScorerHost)")
scorer_body = scorer_body.replace("NewNodeCapacityScorer(svc)", "NewNodeCapacityScorer(host)")
scorer_body = scorer_body.replace("NewGlobalRegionTrafficScorer(svc)", "NewGlobalRegionTrafficScorer(host)")
scorer_body = scorer_body.replace("NodeCapacityScorerWorker", "CapacityScorerWorker")
scorer_body = scorer_body.replace("GlobalRegionTrafficScorerWorker", "GlobalTrafficScorerWorker")
write(
    ROOT / "internal/nodeadmin/worker_scorer.go",
    scorer_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# nodeadmin metrics snapshot
snap_body = chunk(1097, 1224)
snap_body = snap_body.replace("*Service", "MetricsHost")
snap_body = snap_body.replace("svc *Service", "host MetricsHost")
snap_body = snap_body.replace("w.svc", "w.host")
snap_body = snap_body.replace("svc.GetPool()", "host.Pool()")
snap_body = snap_body.replace("NewNodeMetricsSnapshotWorker(svc *Service)", "NewMetricsSnapshotWorker(host MetricsHost)")
snap_body = snap_body.replace("NodeMetricsSnapshotWorker", "MetricsSnapshotWorker")
snap_body = snap_body.replace("w.host.withPostgresLow", "w.host.WithPostgresLow")
write(
    ROOT / "internal/nodeadmin/worker_metrics_snapshot.go",
    snap_body,
    'import (\n\t"context"\n\t"fmt"\n\t"log/slog"\n\t"time"\n\n\tdb "ad-event-processor/internal/domain/db"\n\n\t"github.com/jackc/pgx/v5/pgtype"\n\t"github.com/jackc/pgx/v5/pgxpool"\n)',
)

# nodeadmin metrics worker
metrics_body = chunk(1411, 1601)
metrics_body = metrics_body.replace("*Service", "MetricsHost")
metrics_body = metrics_body.replace("svc *Service", "host MetricsHost")
metrics_body = metrics_body.replace("w.svc", "w.host")
metrics_body = metrics_body.replace("svc.GetPool()", "host.Pool()")
metrics_body = metrics_body.replace("svc != nil && svc.cfg != nil && svc.cfg.NodeID", "host != nil")
metrics_body = metrics_body.replace('nodeID = svc.cfg.NodeID', "nodeID, role, region = host.NodeIdentity()")
metrics_body = metrics_body.replace("NewNodeMetricsWorker(svc *Service)", "NewMetricsWorker(host MetricsHost)")
metrics_body = metrics_body.replace("NodeMetricsWorker", "MetricsWorker")
metrics_body = metrics_body.replace("w.host.withPostgresLow", "w.host.WithPostgresLow")
# fix constructor - replace manual cfg reads with NodeIdentity
old_ctor = """func NewMetricsWorker(host MetricsHost) *MetricsWorker {
\tnodeID, _ := os.Hostname()
\tif host != nil {
\t\tnodeID, role, region = host.NodeIdentity()"
"""
# Simpler: rewrite NewMetricsWorker manually in fix pass
write(
    ROOT / "internal/nodeadmin/worker_metrics.go",
    metrics_body,
    'import (\n\t"context"\n\t"fmt"\n\t"log/slog"\n\t"math"\n\t"os"\n\t"sort"\n\t"sync"\n\t"time"\n\n\tdb "ad-event-processor/internal/domain/db"\n\n\t"github.com/jackc/pgx/v5/pgtype"\n\t"github.com/jackc/pgx/v5/pgxpool"\n)',
)

# campaign worker loops
loops_body = chunk(91, 317)
loops_body = loops_body.replace("*Service", "LoopHost")
loops_body = loops_body.replace("svc *Service", "host LoopHost")
loops_body = loops_body.replace("w.svc", "w.host")
loops_body = loops_body.replace("ErrPostgresGateRejected", "reconciliation.ErrPostgresGateRejected")
write(
    ROOT / "internal/campaign/worker_loops.go",
    loops_body,
    'import (\n\t"context"\n\t"errors"\n\t"log/slog"\n\t"time"\n\n\t"ad-event-processor/internal/database"\n\t"ad-event-processor/internal/domain"\n\t"ad-event-processor/internal/reconciliation"\n)',
)

# campaign drain
drain_body = chunk(708, 782)
drain_body = replace(drain_body, [("*Service", "DrainHost"), ("svc *Service", "host DrainHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/campaign/worker_drain.go",
    drain_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# rtb floor optimizer
floor_body = chunk(372, 419)
floor_body = replace(floor_body, [("*Service", "FloorHost"), ("svc *Service", "host FloorHost"), ("w.svc", "w.host")])
floor_body = floor_body.replace("func (s *Service) StartFloorOptimizerWorker", "func StartFloorOptimizerLoop")
floor_body = floor_body.replace("(s *Service)", "(host FloorHost)")
floor_body = floor_body.replace("s.cfg", "host.FloorOptimizerConfig()")
floor_body = floor_body.replace("NewFloorOptimizerWorker(s, interval)", "NewFloorOptimizerWorker(host, interval)")
write(
    ROOT / "internal/rtbadmin/worker_floor.go",
    floor_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# fraud blacklist janitor
bl_body = chunk(421, 481)
bl_body = replace(bl_body, [("*Service", "BlacklistJanitorHost"), ("svc *Service", "host BlacklistJanitorHost"), ("j.svc", "j.host")])
bl_body = bl_body.replace("GetPool()", "Pool()")
write(
    ROOT / "internal/fraudadmin/worker_blacklist.go",
    bl_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n\n\tdb "ad-event-processor/internal/domain/db"\n)',
)

# billing: usage flush + ledger invariant
billing_body = chunk(38, 48) + chunk(483, 637)
billing_body = billing_body.replace("workerBatchTimeout", "batchTimeout")
billing_body = billing_body.replace("func workerContext", "func batchContext")
billing_body = billing_body.replace("LedgerInvariantAlerter", "InvariantAlerter")
write(
    ROOT / "internal/billingadmin/workers_ledger.go",
    billing_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n\n\t"ad-event-processor/internal/config"\n\n\t"github.com/jackc/pgx/v5/pgxpool"\n)',
)

# privacy erasure + consent + events retention
privacy_body = chunk(319, 370) + chunk(639, 706)
privacy_body = privacy_body.replace("workerContext", "batchContext")
privacy_body = privacy_body.replace("workerBatchTimeout", "batchTimeout")
privacy_body = replace(privacy_body, [("*Service", "Host"), ("svc *Service", "host Host"), ("w.svc", "w.host")])
write(
    ROOT / "internal/privacyadmin/workers_retention.go",
    privacy_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n\n\t"github.com/jackc/pgx/v5/pgxpool"\n)',
)

# credit scoring
credit_body = chunk(784, 874)
credit_body = replace(credit_body, [("*Service", "CreditHost"), ("svc *Service", "host CreditHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/billingadmin/worker_credit.go",
    credit_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# supply audit
supply_body = chunk(876, 959)
supply_body = replace(supply_body, [("*Service", "AuditHost"), ("svc *Service", "host AuditHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/supply/worker_audit.go",
    supply_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# platform system state
sys_body = chunk(178, 209)
sys_body = replace(sys_body, [("*Service", "SystemHost"), ("svc *Service", "host SystemHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/platformadmin/worker_system.go",
    sys_body,
    'import (\n\t"context"\n\t"log/slog"\n\t"time"\n)',
)

# platform nginx
nginx_body = chunk(971, 1095)
nginx_body = replace(nginx_body, [("*Service", "NginxHost"), ("svc *Service", "host NginxHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/platformadmin/worker_nginx.go",
    nginx_body,
    'import (\n\t"context"\n\t"fmt"\n\t"log/slog"\n\t"os"\n\t"path/filepath"\n\t"time"\n)',
)

# platform audit export
audit_body = chunk(1213, 1409)
audit_body = replace(audit_body, [("*Service", "AuditExportHost"), ("svc *Service", "host AuditExportHost"), ("w.svc", "w.host")])
write(
    ROOT / "internal/platformadmin/worker_audit_export.go",
    audit_body,
    'import (\n\t"bufio"\n\t"context"\n\t"encoding/csv"\n\t"fmt"\n\t"log/slog"\n\t"os"\n\t"path/filepath"\n\t"time"\n)',
)

# volume meter
vol_body = chunk(1603, 1855)
vol_body = vol_body.replace("workerContext", "batchContext")
vol_body = vol_body.replace("workerBatchTimeout", "batchTimeout")
vol_body = vol_body.replace("incrementUsageMeterSQL", "incrementUsageMeterSQL")
vol_body = vol_body.replace("*PostgresGate", "*billingadmin.PostgresGate")
write(
    ROOT / "internal/billingadmin/worker_volume.go",
    vol_body,
    'import (\n\t"context"\n\t"fmt"\n\t"log/slog"\n\t"time"\n\n\t"ad-event-processor/internal/database"\n\n\t"github.com/jackc/pgx/v5/pgxpool"\n)',
)

# fraud ml sync
ml_body = chunk(1857, 2114)
ml_body = replace(ml_body, [("*Service", "MLSyncHost"), ("svc *Service", "host MLSyncHost"), ("o.svc", "o.host"), ("w.svc", "w.host")])
write(
    ROOT / "internal/fraudadmin/worker_ml_sync.go",
    ml_body,
    'import (\n\t"context"\n\t"fmt"\n\t"log/slog"\n\t"time"\n)',
)

# operation lease block from workers.go
op_block = chunk(2495, len(lines))
OP_LEASE.write_text(OP_LEASE.read_text().rstrip() + "\n\n" + op_block)

# slim workers.go: header + common + TLS + recon
header = chunk(1, 36)
common = chunk(38, 61)
tls = chunk(63, 89)
recon = chunk(961, 969)
WORKERS.write_text(header + common + "\n" + tls + "\n" + recon)

# worker aliases stub
ALIASES.write_text(
    """package controlplane

import (
\t"ad-event-processor/internal/billingadmin"
\t"ad-event-processor/internal/campaign"
\t"ad-event-processor/internal/fraudadmin"
\t"ad-event-processor/internal/nodeadmin"
\t"ad-event-processor/internal/platformadmin"
\t"ad-event-processor/internal/privacyadmin"
\t"ad-event-processor/internal/reconciliation"
\t"ad-event-processor/internal/rtbadmin"
\t"ad-event-processor/internal/supply"
)

type (
\tNodeCapacityScorerWorker       = nodeadmin.CapacityScorerWorker
\tGlobalRegionTrafficScorerWorker = nodeadmin.GlobalTrafficScorerWorker
\tNodeMetricsSnapshotWorker     = nodeadmin.MetricsSnapshotWorker
\tNodeMetricsWorker             = nodeadmin.MetricsWorker
\tScheduleWorker                 = campaign.ScheduleWorker
\tPacingControllerWorker         = campaign.PacingControllerWorker
\tAutoscaleBudgetWorker          = campaign.AutoscaleBudgetWorker
\tDeliveryOptimizerWorker        = campaign.DeliveryOptimizerWorker
\tCampaignDrainWorker            = campaign.CampaignDrainWorker
\tFloorOptimizerWorker           = rtbadmin.FloorOptimizerWorker
\tBlacklistJanitor                = fraudadmin.BlacklistJanitor
\tUsageDailyFlushWorker          = billingadmin.UsageDailyFlushWorker
\tLedgerInvariantWorker          = billingadmin.LedgerInvariantWorker
\tVolumeMeterWorker              = billingadmin.VolumeMeterWorker
\tCreditScoringWorker            = billingadmin.CreditScoringWorker
\tErasureWorker                  = privacyadmin.ErasureWorker
\tConsentRetentionWorker         = privacyadmin.ConsentRetentionWorker
\tEventsRetentionWorker          = privacyadmin.EventsRetentionWorker
\tSupplyAuditWorker               = supply.AuditWorker
\tSystemStateWorker               = platformadmin.SystemStateWorker
\tNginxConfigWorker               = platformadmin.NginxConfigWorker
\tAuditExportWorker               = platformadmin.AuditExportWorker
\tFraudModelSyncWorker            = fraudadmin.MLSyncWorker
\tFraudModelSyncOrchestrator      = fraudadmin.MLSyncOrchestrator
\tReconWorker                     = reconciliation.ReconWorker
)
"""
)

print("extracted; workers.go lines:", len(WORKERS.read_text().splitlines()))
