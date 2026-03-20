#!/usr/bin/env bash
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "请使用 root 运行此脚本。" >&2
  exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

APP_DIR="${APP_DIR:-${PROJECT_DIR}}"
CONFIG_DIR="${CONFIG_DIR:-/etc/proxyrelay}"
STATE_DIR="${STATE_DIR:-/var/lib/proxyrelay}"
RUNTIME_DIR="${RUNTIME_DIR:-${STATE_DIR}/runtime}"
SERVICE_DIR="${SERVICE_DIR:-/etc/systemd/system}"
PROXYRELAY_USER="${PROXYRELAY_USER:-proxyrelay}"
PROXYRELAY_GROUP="${PROXYRELAY_GROUP:-${PROXYRELAY_USER}}"
MIHOMO_BIN="${MIHOMO_BIN:-/usr/local/bin/mihomo}"
CONFIG_FILE="${CONFIG_DIR}/proxyrelay.yaml"

require_file() {
  local target="$1"
  if [[ ! -f "${target}" ]]; then
    echo "缺少文件: ${target}" >&2
    exit 1
  fi
}

render_template() {
  local source_file="$1"
  local target_file="$2"

  sed \
    -e "s#/opt/proxyrelay#${APP_DIR}#g" \
    -e "s#/etc/proxyrelay#${CONFIG_DIR}#g" \
    -e "s#/var/lib/proxyrelay#${STATE_DIR}#g" \
    -e "s#/usr/local/bin/mihomo#${MIHOMO_BIN}#g" \
    "${source_file}" > "${target_file}"
}

require_file "${PROJECT_DIR}/examples/proxyrelay.yaml"
require_file "${PROJECT_DIR}/deploy/proxyrelayd.service"
require_file "${PROJECT_DIR}/deploy/mihomo.service"
require_file "${APP_DIR}/src/index.js"
require_file "${APP_DIR}/package.json"
require_file "${APP_DIR}/frontend/package.json"

if ! getent group "${PROXYRELAY_GROUP}" >/dev/null 2>&1; then
  groupadd --system "${PROXYRELAY_GROUP}"
fi

if ! id "${PROXYRELAY_USER}" >/dev/null 2>&1; then
  useradd --system --home "${APP_DIR}" --shell /usr/sbin/nologin --gid "${PROXYRELAY_GROUP}" "${PROXYRELAY_USER}"
fi

mkdir -p "${APP_DIR}" "${CONFIG_DIR}" "${RUNTIME_DIR}" "${STATE_DIR}/logs" "${SERVICE_DIR}"
chown -R "${PROXYRELAY_USER}:${PROXYRELAY_GROUP}" "${CONFIG_DIR}" "${STATE_DIR}"
chmod 700 "${CONFIG_DIR}" "${STATE_DIR}"

if [[ ! -f "${CONFIG_FILE}" ]]; then
  sed \
    -e "s#workdir: /var/lib/proxyrelay/runtime#workdir: ${RUNTIME_DIR}#g" \
    -e "s#/usr/local/bin/mihomo#${MIHOMO_BIN}#g" \
    "${PROJECT_DIR}/examples/proxyrelay.yaml" > "${CONFIG_FILE}"
fi

chown "${PROXYRELAY_USER}:${PROXYRELAY_GROUP}" "${CONFIG_FILE}"
chmod 600 "${CONFIG_FILE}"

# Build frontend UI
echo "正在构建前端资源..."
cd "${APP_DIR}" && npm run build:ui

render_template "${PROJECT_DIR}/deploy/proxyrelayd.service" "${SERVICE_DIR}/proxyrelayd.service"
render_template "${PROJECT_DIR}/deploy/mihomo.service" "${SERVICE_DIR}/mihomo.service"
chmod 644 "${SERVICE_DIR}/proxyrelayd.service" "${SERVICE_DIR}/mihomo.service"

if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
fi

cat <<EOF
ProxyRelay Ubuntu 安装准备已完成。

当前路径:
- 应用目录: ${APP_DIR}
- 配置文件: ${CONFIG_FILE}
- 运行目录: ${RUNTIME_DIR}
- Mihomo 二进制: ${MIHOMO_BIN}

下一步建议:
1. 编辑 ${CONFIG_FILE}，替换真实订阅、密码和 secret。
2. 确认 Mihomo 已安装且可执行: ${MIHOMO_BIN}
3. 运行预检:
   sudo -u ${PROXYRELAY_USER} env PROXYRELAY_CONFIG=${CONFIG_FILE} node ${APP_DIR}/src/index.js preflight
4. 启动服务:
   sudo systemctl enable --now proxyrelayd
   sudo systemctl enable --now mihomo
5. 查看日志:
   journalctl -u proxyrelayd -f
   journalctl -u mihomo -f
EOF
