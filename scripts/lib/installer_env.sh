#!/usr/bin/env bash

installer_read_env() {
  local key="$1"
  local file val
  for file in "${ROOT:-.}/.env" "${ROOT:-.}/deploy/installer/install.env"; do
    if [[ -f "$file" ]]; then
      val="$(grep -m1 "^${key}=" "$file" 2> /dev/null | cut -d= -f2- || true)"
      if [[ -n "$val" ]]; then
        echo "$val"
        return 0
      fi
    fi
  done
  echo ""
}

installer_env_dual() {
  local ad_key="$1"
  local legacy_key="$2"
  local val
  val="$(installer_read_env "$ad_key")"
  if [[ -n "$val" ]]; then
    echo "$val"
    return 0
  fi
  installer_read_env "$legacy_key"
}

installer_use_release_images() {
  local flag img
  flag="$(installer_env_dual AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES)"
  img="$(installer_env_dual AD_EVENT_PROCESSOR_APP_IMAGE AD_EVENT_PROCESSOR_APP_IMAGE)"
  [[ "$flag" == "1" ]] || [[ -n "$img" ]]
}

installer_release_app_image() {
  installer_env_dual AD_EVENT_PROCESSOR_APP_IMAGE AD_EVENT_PROCESSOR_APP_IMAGE
}

installer_license_key() {
  installer_env_dual AD_EVENT_PROCESSOR_LICENSE_KEY AD_EVENT_PROCESSOR_LICENSE_KEY
}

installer_license_mode() {
  local v
  v="$(installer_env_dual AD_EVENT_PROCESSOR_LICENSE_MODE AD_EVENT_PROCESSOR_LICENSE_MODE)"
  echo "${v:-file}"
}

installer_license_required() {
  local v
  v="$(installer_env_dual AD_EVENT_PROCESSOR_LICENSE_REQUIRED AD_EVENT_PROCESSOR_LICENSE_REQUIRED)"
  echo "${v:-1}"
}

ad_event_processor_read_env() { installer_read_env "$1"; }
ad_event_processor_use_release_images() { installer_use_release_images; }
