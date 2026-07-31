#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

CMD="${1:-status}"

INFRA=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse)
FULL=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse processor tracker-0 auth management payment billing notifier ivt-detector)
SINGLE_VPS=(db redis-0 redis-1 redis-2 redis-3 clickhouse processor tracker-0 control)
INGEST_ONLY=(db redis-0 redis-1 redis-2 redis-3 processor tracker-0 control)
NETWORK_OPERATOR=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse processor tracker-0 control)
SENTINEL=(redis-0 redis-0-replica sentinel-0 sentinel-1 sentinel-2)

case "$CMD" in
infra | up-infra)
	docker compose up -d "${INFRA[@]}"
	;;
full | up-full)
	echo "stack.sh: full runs single-vps monolith (split_control: use legacy-full)" >&2
	docker compose --profile single_vps up -d "${SINGLE_VPS[@]}"
	;;
legacy-full | up-legacy-full)
	echo "stack.sh: WARN split_control profile is deprecated; use single-vps" >&2
	docker compose --profile split_control up -d "${FULL[@]}"
	;;
single-vps | up-single-vps)
	docker compose --profile single_vps up -d "${SINGLE_VPS[@]}"
	;;
ingest-only | up-ingest-only)
	CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
		CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
		docker compose --profile ingest_only up -d "${INGEST_ONLY[@]}"
	;;
network-operator | up-network-operator)
	docker compose --profile network_operator up -d "${NETWORK_OPERATOR[@]}"
	;;
analytics-ml | up-analytics-ml)
	docker compose --profile analytics_ml --profile fraud-scorer up -d ivt-detector fraud-scorer clickhouse
	;;
sentinel | up-sentinel)
	docker compose up -d "${SENTINEL[@]}"
	;;
multi-region | up-multi-region)
	sub="${2:-up}"
	case "$sub" in
	up)
		docker compose --profile multi-region up -d region-proxy processor-1
		;;
	broker)
		docker compose --profile multi-region up -d broker
		;;
	down)
		docker compose --profile multi-region stop region-proxy broker 2>/dev/null || true
		;;
	status)
		docker compose --profile multi-region ps
		;;
	*)
		echo "usage: $0 multi-region {up|broker|down|status}" >&2
		exit 2
		;;
	esac
	;;
crypto | up-crypto)
	sub="${2:-up}"
	case "$sub" in
	up)
		if [ -f deploy/payment/crypto-sandbox.env ]; then
			set -a
			source deploy/payment/crypto-sandbox.env
			set +a
		fi
		docker compose --profile crypto up -d db-payment payment
		;;
	down)
		docker compose --profile crypto stop payment 2>/dev/null || true
		;;
	status)
		docker compose --profile crypto ps
		;;
	*)
		echo "usage: $0 crypto {up|down|status}" >&2
		exit 2
		;;
	esac
	;;
down)
	docker compose down
	;;
status)
	docker compose ps
	echo "--- multi-region profile ---"
	docker compose --profile multi-region ps
	;;
build)
	docker compose build
	bash "$SCRIPTS/dev/bpf_setup.sh" || echo "dev_stack: WARN bpf_setup failed (optional dev tooling)" >&2
	;;
bpf)
	bash "$SCRIPTS/dev/bpf_setup.sh"
	;;
probe)
	sub="${2:-status}"
	case "$sub" in
	start | stop | status | report)
		sudo bash "$SCRIPTS/dev/bpf_session.sh" "$sub" "${3:-}"
		;;
	*)
		echo "usage: $0 probe {start|stop|status|report} [session_dir]" >&2
		exit 2
		;;
	esac
	;;
*)
	echo "usage: $0 {infra|full|legacy-full|single-vps|ingest-only|network-operator|analytics-ml|sentinel|multi-region|crypto|down|status|build|bpf|probe}" >&2
	exit 2
	;;
esac
