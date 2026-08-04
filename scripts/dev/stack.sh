#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

espx_read_env() {
	local key="$1"
	local file val
	for file in "$ROOT/.env" "$ROOT/deploy/installer/install.env"; do
		if [[ -f "$file" ]]; then
			val="$(grep -m1 "^${key}=" "$file" 2>/dev/null | cut -d= -f2- || true)"
			if [[ -n "$val" ]]; then
				echo "$val"
				return 0
			fi
		fi
	done
	echo ""
}

espx_use_release_images() {
	local flag img
	flag="$(espx_read_env ESPX_USE_RELEASE_IMAGES)"
	img="$(espx_read_env ESPX_APP_IMAGE)"
	[[ "$flag" == "1" ]] || [[ -n "$img" ]]
}

espx_ingress_enabled() {
	[[ "$(espx_read_env INGRESS_ENABLED)" == "1" ]]
}

espx_ingress_enabled() {
	[[ "$(espx_read_env INGRESS_ENABLED)" == "1" ]]
}

espx_compose() {
	local -a env_args=()
	local -a file_args=(-f "$ROOT/docker-compose.yaml")
	if espx_use_release_images; then
		file_args+=(-f "$ROOT/deploy/compose/docker-compose.release.yaml")
	fi
	if [[ -f "$ROOT/.env" ]]; then
		env_args+=(--env-file "$ROOT/.env")
	fi
	if [[ -f "$ROOT/install.compose.env" ]]; then
		env_args+=(--env-file "$ROOT/install.compose.env")
	fi
	docker compose "${file_args[@]}" "${env_args[@]}" "$@"
}

CMD="${1:-status}"

INFRA=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse)
SINGLE_VPS=(db redis-0 redis-1 redis-2 redis-3 clickhouse processor tracker-0 control)
INGEST_ONLY=(db redis-0 redis-1 redis-2 redis-3 processor tracker-0 control)
NETWORK_OPERATOR=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse processor tracker-0 control)
SENTINEL=(redis-0 redis-0-replica sentinel-0 sentinel-1 sentinel-2)

case "$CMD" in
infra | up-infra)
	espx_compose up -d "${INFRA[@]}"
	;;
full | up-full)
	echo "stack.sh: full runs single-vps monolith" >&2
	espx_compose --profile single_vps up -d "${SINGLE_VPS[@]}"
	;;
legacy-full | up-legacy-full)
	echo "stack.sh: legacy-full removed; use single-vps or network-operator" >&2
	exit 1
	;;
single-vps | up-single-vps)
	local -a prof=(--profile single_vps)
	if espx_ingress_enabled; then
		prof+=(--profile ingress)
		bash "$SCRIPTS/install/render_ingress.sh"
	fi
	if espx_use_release_images; then
		espx_compose "${prof[@]}" pull tracker-0 processor control
		espx_compose "${prof[@]}" up -d --no-build "${SINGLE_VPS[@]}"
	else
		espx_compose "${prof[@]}" up -d "${SINGLE_VPS[@]}"
	fi
	;;
ingest-only | up-ingest-only)
	CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
		CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
		espx_compose --profile ingest_only up -d "${INGEST_ONLY[@]}"
	;;
network-operator | up-network-operator)
	espx_compose --profile network_operator up -d "${NETWORK_OPERATOR[@]}"
	;;
analytics-ml | up-analytics-ml)
	espx_compose --profile analytics_ml --profile fraud-scorer up -d ivt-detector fraud-scorer clickhouse
	;;
sentinel | up-sentinel)
	espx_compose up -d "${SENTINEL[@]}"
	;;
multi-region | up-multi-region)
	sub="${2:-up}"
	case "$sub" in
	up)
		espx_compose --profile multi-region up -d region-proxy processor-1
		;;
	broker)
		espx_compose --profile multi-region up -d broker
		;;
	down)
		espx_compose --profile multi-region stop region-proxy broker 2>/dev/null || true
		;;
	status)
		espx_compose --profile multi-region ps
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
		espx_compose --profile crypto up -d db-payment control
		;;
	down)
		espx_compose --profile crypto stop control 2>/dev/null || true
		;;
	status)
		espx_compose --profile crypto ps
		;;
	*)
		echo "usage: $0 crypto {up|down|status}" >&2
		exit 2
		;;
	esac
	;;
down)
	espx_compose down
	;;
status)
	espx_compose ps
	echo "--- multi-region profile ---"
	espx_compose --profile multi-region ps
	;;
build)
	if espx_use_release_images; then
		echo "stack.sh: pulling release images (ESPX_USE_RELEASE_IMAGES / ESPX_APP_IMAGE)" >&2
		espx_compose pull tracker-0 processor control
	else
		espx_compose build
	fi
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
