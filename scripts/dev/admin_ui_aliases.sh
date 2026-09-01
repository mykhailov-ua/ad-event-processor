# shellcheck shell=bash
# Source from repo root: source scripts/dev/admin_ui_aliases.sh
#
# Adds scripts/dev to PATH (aed-admin wrapper) and defines:
#   aed-admin up|down|status|control|web|logs
#   aed-admin-up, aed-admin-down, aed-admin-status (shortcuts)
#
# One-liner without sourcing:
#   bash scripts/dev/aed-admin up

if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
  _aed_admin_ui_aliases_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
elif [[ -n "${ZSH_VERSION:-}" ]]; then
  _aed_admin_ui_aliases_dir="$(cd "$(dirname "${(%):-%x}")" && pwd)"
else
  _aed_admin_ui_aliases_dir="$(cd "$(dirname "$0")" && pwd)"
fi

case ":${PATH}:" in
  *":${_aed_admin_ui_aliases_dir}:"*) ;;
  *) export PATH="${_aed_admin_ui_aliases_dir}:${PATH}" ;;
esac

_aed_admin_ui_script="$_aed_admin_ui_aliases_dir/admin_ui.sh"

aed-admin() {
  bash "$_aed_admin_ui_script" "$@"
}

aed-admin-up() {
  aed-admin up
}

aed-admin-down() {
  aed-admin down
}

aed-admin-status() {
  aed-admin status
}

aed-admin-web() {
  aed-admin web
}

aed-admin-control() {
  aed-admin control
}

aed-admin-stack() {
  aed-admin stack
}

aed-admin-seed() {
  aed-admin seed
}

unset _aed_admin_ui_aliases_dir _aed_admin_ui_script
