#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_CONFIG="${ROOT_DIR}/config/proxyrelay.yaml"
RUNTIME_DIR="${ROOT_DIR}/runtime/local-run"
CONFIG_FILE="${RUNTIME_DIR}/proxyrelay.local.yaml"
MIHOMO_BIN="${MIHOMO_BIN:-/usr/local/bin/mihomo}"
SERVER_HOST="${SERVER_HOST:-0.0.0.0}"
SERVER_PORT="${SERVER_PORT:-8787}"
CONTROLLER_HOST="${CONTROLLER_HOST:-127.0.0.1}"
CONTROLLER_PORT="${CONTROLLER_PORT:-11235}"
CONTROLLER_SECRET="${CONTROLLER_SECRET:-proxyrelay-local-secret}"

MIHOMO_PID=""

step() {
  printf '\n[%s/6] %s\n' "$1" "$2"
}

require_cmd() {
  local bin="$1"
  if ! command -v "${bin}" >/dev/null 2>&1; then
    printf 'missing command: %s\n' "${bin}" >&2
    exit 1
  fi
}

cleanup() {
  local exit_code=$?
  if [[ -n "${MIHOMO_PID}" ]] && kill -0 "${MIHOMO_PID}" >/dev/null 2>&1; then
    kill "${MIHOMO_PID}" >/dev/null 2>&1 || true
    wait "${MIHOMO_PID}" >/dev/null 2>&1 || true
  fi
  exit "${exit_code}"
}

wait_for_controller() {
  local attempts=20
  local index
  for ((index = 1; index <= attempts; index += 1)); do
    if curl -fsS \
      -H "Authorization: Bearer ${CONTROLLER_SECRET}" \
      "http://${CONTROLLER_HOST}:${CONTROLLER_PORT}/version" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

kill_residual() {
  pkill -9 -x mihomo 2>/dev/null || true
  pkill -9 -f "node src/index.js" 2>/dev/null || true
  sleep 1
}

check_ports() {
  local busy
  busy="$(ss -lnt "( sport = :${SERVER_PORT} or sport = :${CONTROLLER_PORT} or sport = :10801 or sport = :10802 or sport = :10803 )" | sed '1d')"
  if [[ -n "${busy}" ]]; then
    printf 'ports busy, killing residual processes...\n'
    kill_residual
    busy="$(ss -lnt "( sport = :${SERVER_PORT} or sport = :${CONTROLLER_PORT} or sport = :10801 or sport = :10802 or sport = :10803 )" | sed '1d')"
    if [[ -n "${busy}" ]]; then
      printf 'ports still busy after cleanup:\n%s\n' "${busy}" >&2
      exit 1
    fi
    printf 'ports freed, continuing...\n'
  fi
}

trap cleanup EXIT INT TERM

step 1 "check dependencies"
require_cmd node
require_cmd curl
require_cmd perl
require_cmd sed
require_cmd ss

if [[ ! -f "${SOURCE_CONFIG}" ]]; then
  printf 'missing config template: %s\n' "${SOURCE_CONFIG}" >&2
  exit 1
fi

if [[ ! -x "${MIHOMO_BIN}" ]]; then
  printf 'mihomo is not executable: %s\n' "${MIHOMO_BIN}" >&2
  exit 1
fi

step 2 "stop systemd services"
if command -v sudo >/dev/null 2>&1; then
  sudo systemctl stop proxyrelayd mihomo >/dev/null 2>&1 || true
fi

step 3 "prepare local managed config"
mkdir -p "${RUNTIME_DIR}"
if [[ ! -f "${CONFIG_FILE}" ]]; then
  cp "${SOURCE_CONFIG}" "${CONFIG_FILE}"
  printf 'created local config from template\n'
else
  printf 'reusing existing local config\n'
fi
perl -0pi -e '
  s/^  host:.*$/  host: '"${SERVER_HOST}"'/m;
  s/^  port:.*$/  port: '"${SERVER_PORT}"'/m;
  s#^  workdir:.*$#  workdir: '"${RUNTIME_DIR}"'#m;
  s#^  mihomo_binary:.*$#  mihomo_binary: '"${MIHOMO_BIN}"'#m;
  s#^  external_controller:.*$#  external_controller: http://'"${CONTROLLER_HOST}"':'"${CONTROLLER_PORT}"'#m;
  s#^  external_secret:.*$#  external_secret: '"${CONTROLLER_SECRET}"'#m;
  s/^  render_only:.*$/  render_only: false/m;
' "${CONFIG_FILE}"

step 4 "build frontend and render local runtime files"
(cd "${ROOT_DIR}" && npm run build:ui >/dev/null 2>&1)
PROXYRELAY_CONFIG="${CONFIG_FILE}" node src/index.js preflight >/dev/null 2>&1 || true
check_ports

step 5 "start mihomo"
stdbuf -oL -eL "${MIHOMO_BIN}" \
  -d "${RUNTIME_DIR}" \
  -f "${RUNTIME_DIR}/mihomo.yaml" \
  > >(sed 's/^/[mihomo] /') 2>&1 &
MIHOMO_PID="$!"

if ! wait_for_controller; then
  printf 'controller did not become ready on %s:%s\n' "${CONTROLLER_HOST}" "${CONTROLLER_PORT}" >&2
  exit 1
fi

step 6 "start axis"
printf 'config file: %s\n' "${CONFIG_FILE}"
printf 'login url : http://%s:%s/login\n' "${SERVER_HOST}" "${SERVER_PORT}"
printf 'press Ctrl+C to stop both processes\n\n'

exec env PROXYRELAY_CONFIG="${CONFIG_FILE}" \
  stdbuf -oL -eL node src/index.js serve \
  > >(sed 's/^/[axis] /') 2>&1
