#!/usr/bin/env bash
# Helpers for DNS lab (/etc/hosts -> 127.0.0.1, no public DNS).
dns_lab_hosts_resolve() {
  local host="$1"
  getent ahosts "$host" 2> /dev/null | awk '{print $1}' | head -1
}

dns_lab_hosts_has_loopback() {
  local host="$1"
  local ip
  ip="$(dns_lab_hosts_resolve "$host")"
  [[ "$ip" == "127.0.0.1" || "$ip" == "::1" ]]
}

dns_lab_hosts_install() {
  local hosts=("$@")
  local missing=()
  for h in "${hosts[@]}"; do
    if ! dns_lab_hosts_has_loopback "$h"; then
      missing+=("$h")
    fi
  done
  if ((${#missing[@]} == 0)); then
    return 0
  fi
  if [[ "${DNS_LAB_INSTALL_HOSTS:-}" != "1" ]]; then
    printf 'dns-lab: add to /etc/hosts:\n  127.0.0.1 %s\n' "${missing[*]}" >&2
    printf 'dns-lab: or re-run with DNS_LAB_INSTALL_HOSTS=1 (requires sudo)\n' >&2
    return 1
  fi
  local line="127.0.0.1 ${missing[*]}"
  if grep -qF "$line" /etc/hosts 2> /dev/null; then
    return 0
  fi
  if [[ "$(id -u)" -eq 0 ]]; then
    printf '%s\n' "$line" >> /etc/hosts
    return 0
  fi
  if command -v sudo > /dev/null 2>&1; then
    printf '%s\n' "$line" | sudo tee -a /etc/hosts > /dev/null
    return 0
  fi
  printf 'dns-lab: cannot write /etc/hosts (no sudo)\n' >&2
  return 1
}
