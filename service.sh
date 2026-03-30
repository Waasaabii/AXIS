#!/usr/bin/env bash
# ============================================================================
#  AXIS (Anycast X-Proxy Integration System) — systemd 服务管理脚本
#  用法: sudo ./service.sh {install|uninstall|start|stop|restart|status|logs}
# ============================================================================
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

APP_DIR="${APP_DIR:-${ROOT_DIR}}"
CONFIG_DIR="${CONFIG_DIR:-/etc/proxyrelay}"
STATE_DIR="${STATE_DIR:-/var/lib/proxyrelay}"
RUNTIME_DIR="${STATE_DIR}/runtime"
SERVICE_DIR="/etc/systemd/system"
PROXYRELAY_USER="${PROXYRELAY_USER:-${SUDO_USER:-root}}"
PROXYRELAY_GROUP="${PROXYRELAY_GROUP:-${PROXYRELAY_USER}}"
MIHOMO_BIN="${MIHOMO_BIN:-/usr/local/bin/mihomo}"
MIHOMO_VERSION="${MIHOMO_VERSION:-v1.19.21}"
CONFIG_FILE="${CONFIG_DIR}/proxyrelay.yaml"

# ── Pretty output ─────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { printf "${CYAN}[▸]${NC} %s\n" "$*"; }
ok()   { printf "${GREEN}[✔]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[!]${NC} %s\n" "$*"; }
err()  { printf "${RED}[✘]${NC} %s\n" "$*" >&2; }

# ── Architecture ──────────────────────────────────────────────────────────

detect_arch() {
  local arch
  arch="$(uname -m)"
  case "${arch}" in
    x86_64)  echo "amd64" ;;
    aarch64) echo "arm64" ;;
    armv7l)  echo "armv7" ;;
    *)       echo "${arch}" ;;
  esac
}

# ── require root ──────────────────────────────────────────────────────────

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    err "请使用 sudo 或 root 用户运行此脚本"
    exit 1
  fi
}

# ── Dependency helpers ────────────────────────────────────────────────────

ensure_nodejs() {
  if command -v node >/dev/null 2>&1; then
    ok "Node.js 已安装: $(node --version)"
    return
  fi
  log "安装 Node.js 18.x LTS ..."
  if command -v apt-get >/dev/null 2>&1; then
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
    apt-get install -y nodejs
  elif command -v yum >/dev/null 2>&1; then
    curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -
    yum install -y nodejs
  else
    err "不支持的包管理器，请手动安装 Node.js >= 18"
    exit 1
  fi
  ok "Node.js 已安装: $(node --version)"
}

ensure_pnpm() {
  if command -v pnpm >/dev/null 2>&1; then
    ok "pnpm 已安装: $(pnpm --version)"
    return
  fi
  if ! command -v corepack >/dev/null 2>&1; then
    err "未找到 corepack，无法安装 pnpm"
    exit 1
  fi
  log "安装 pnpm ..."
  corepack enable
  corepack prepare pnpm@10.32.1 --activate
  ok "pnpm 已安装: $(pnpm --version)"
}

ensure_mihomo() {
  if [[ -x "${MIHOMO_BIN}" ]]; then
    ok "mihomo 已安装: $("${MIHOMO_BIN}" -v 2>&1 | head -1)"
    return
  fi
  local arch
  arch="$(detect_arch)"
  local filename="mihomo-linux-${arch}-${MIHOMO_VERSION}.gz"
  local url="https://github.com/MetaCubeX/mihomo/releases/download/${MIHOMO_VERSION}/${filename}"
  log "下载 mihomo ${MIHOMO_VERSION} (${arch}) ..."
  local tmp_gz
  tmp_gz="$(mktemp)"
  curl -fsSL -o "${tmp_gz}" "${url}" || { err "下载 mihomo 失败"; rm -f "${tmp_gz}"; exit 1; }
  gunzip -f "${tmp_gz}"
  local tmp_bin="${tmp_gz%.gz}"
  [[ ! -f "${tmp_bin}" ]] && tmp_bin="${tmp_gz}"
  mv "${tmp_bin}" "${MIHOMO_BIN}"
  chmod +x "${MIHOMO_BIN}"
  ok "mihomo 已安装到 ${MIHOMO_BIN}"
}

# ── render service template ───────────────────────────────────────────────

render_template() {
  sed \
    -e "s#/opt/proxyrelay#${APP_DIR}#g" \
    -e "s#/etc/proxyrelay#${CONFIG_DIR}#g" \
    -e "s#/var/lib/proxyrelay#${STATE_DIR}#g" \
    -e "s#/usr/local/bin/mihomo#${MIHOMO_BIN}#g" \
    -e "s#User=proxyrelay#User=${PROXYRELAY_USER}#g" \
    -e "s#Group=proxyrelay#Group=${PROXYRELAY_GROUP}#g" \
    "$1" > "$2"
}

# ============================================================================
#  Commands
# ============================================================================

cmd_install() {
  require_root
  printf "\n${CYAN}══════════════════════════════════════════════════${NC}\n"
  printf "${CYAN}  ProxyRelay 系统服务安装${NC}\n"
  printf "${CYAN}══════════════════════════════════════════════════${NC}\n\n"

  # ── 1. dependencies
  log "检查系统依赖..."
  ensure_nodejs
  ensure_pnpm
  ensure_mihomo

  for cmd in curl ss; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      apt-get install -y "${cmd}" >/dev/null 2>&1 || true
    fi
  done

  # ── 2. system user & dirs
  log "配置系统用户与目录..."
  if ! getent group "${PROXYRELAY_GROUP}" >/dev/null 2>&1; then
    groupadd --system "${PROXYRELAY_GROUP}" || true
  fi
  if ! id "${PROXYRELAY_USER}" >/dev/null 2>&1; then
    useradd --system --home "${APP_DIR}" --shell /usr/sbin/nologin --gid "${PROXYRELAY_GROUP}" "${PROXYRELAY_USER}" || true
  fi
  ok "用户 ${PROXYRELAY_USER} 就绪"

  mkdir -p "${APP_DIR}" "${CONFIG_DIR}" "${RUNTIME_DIR}" "${STATE_DIR}/logs"
  chown -R "${PROXYRELAY_USER}:${PROXYRELAY_GROUP}" "${CONFIG_DIR}" "${STATE_DIR}"
  chmod 700 "${CONFIG_DIR}" "${STATE_DIR}"
  ok "目录结构就绪"

  # ── 3. config
  local example_src="${ROOT_DIR}/config/proxyrelay.example.yaml"
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    if [[ -f "${example_src}" ]]; then
      cp "${example_src}" "${CONFIG_FILE}"
    elif [[ -f "${ROOT_DIR}/examples/proxyrelay.yaml" ]]; then
      cp "${ROOT_DIR}/examples/proxyrelay.yaml" "${CONFIG_FILE}"
    else
      err "缺少配置模板文件"
      exit 1
    fi
    # adjust paths for system deployment
    perl -0pi -e '
      s#^  workdir:.*$#  workdir: '"${RUNTIME_DIR}"'#m;
      s#^  mihomo_binary:.*$#  mihomo_binary: '"${MIHOMO_BIN}"'#m;
      s/^  render_only:.*$/  render_only: false/m;
      s/^  password_hash:.*$/  password_hash: ""/m;
      s/^  password:.*$/  password: ""/m;
    ' "${CONFIG_FILE}"
    log "已生成配置文件 (首次登录请使用 admin/admin)"
  else
    log "配置文件已存在，跳过"
  fi
  chown "${PROXYRELAY_USER}:${PROXYRELAY_GROUP}" "${CONFIG_FILE}"
  chmod 600 "${CONFIG_FILE}"
  ok "配置就绪: ${CONFIG_FILE}"

  # ── 4. npm deps & frontend build
  log "安装项目依赖..."
  (cd "${APP_DIR}" && pnpm install --frozen-lockfile >/dev/null 2>&1)
  log "构建前端..."
  (cd "${APP_DIR}" && pnpm run build:ui >/dev/null 2>&1)
  chown -R "${PROXYRELAY_USER}:${PROXYRELAY_GROUP}" "${APP_DIR}/frontend/dist" 2>/dev/null || true
  ok "项目构建完成"

  # ── 5. render initial runtime config
  log "渲染初始运行态配置..."
  sudo -u "${PROXYRELAY_USER}" env PROXYRELAY_CONFIG="${CONFIG_FILE}" \
    node "${APP_DIR}/src/index.js" preflight >/dev/null 2>&1 || true
  ok "运行态就绪"

  # ── 6. install systemd services
  log "安装 systemd 服务..."
  render_template "${ROOT_DIR}/deploy/proxyrelayd.service" "${SERVICE_DIR}/proxyrelayd.service"
  render_template "${ROOT_DIR}/deploy/mihomo.service" "${SERVICE_DIR}/mihomo.service"
  chmod 644 "${SERVICE_DIR}/proxyrelayd.service" "${SERVICE_DIR}/mihomo.service"
  systemctl daemon-reload
  ok "服务已注册"

  printf "\n${GREEN}══════════════════════════════════════════════════${NC}\n"
  printf "${GREEN}  安装完成！${NC}\n"
  printf "${GREEN}══════════════════════════════════════════════════${NC}\n\n"
  printf "  配置文件 : ${CONFIG_FILE}\n"
  printf "  应用目录 : ${APP_DIR}\n"
  printf "  运行目录 : ${RUNTIME_DIR}\n"
  printf "  mihomo   : ${MIHOMO_BIN}\n\n"
  printf "  启动服务 : sudo ./service.sh start\n"
  printf "  查看状态 : sudo ./service.sh status\n"
  printf "  查看日志 : sudo ./service.sh logs\n\n"
}

cmd_uninstall() {
  require_root
  log "停止并禁用服务..."
  systemctl stop proxyrelayd mihomo 2>/dev/null || true
  systemctl disable proxyrelayd mihomo 2>/dev/null || true
  rm -f "${SERVICE_DIR}/proxyrelayd.service" "${SERVICE_DIR}/mihomo.service"
  systemctl daemon-reload
  ok "systemd 服务已移除"
  printf "\n  注意: 配置文件 (${CONFIG_FILE}) 和数据 (${STATE_DIR}) 未被删除。\n"
  printf "  如需完全清理请手动删除。\n\n"
}

cmd_start() {
  require_root
  log "启动 ProxyRelay 服务..."
  systemctl start proxyrelayd
  systemctl start mihomo
  ok "服务已启动"
  cmd_status
}

cmd_stop() {
  require_root
  log "停止 ProxyRelay 服务..."
  systemctl stop mihomo 2>/dev/null || true
  systemctl stop proxyrelayd 2>/dev/null || true
  ok "服务已停止"
}

cmd_restart() {
  require_root
  log "重启 ProxyRelay 服务..."
  systemctl restart proxyrelayd
  systemctl restart mihomo
  ok "服务已重启"
  cmd_status
}

cmd_status() {
  printf "\n${CYAN}── proxyrelayd ──${NC}\n"
  if [[ -f "${SERVICE_DIR}/proxyrelayd.service" ]]; then
    systemctl status proxyrelayd --no-pager -l || true
  else
    warn "proxyrelayd 未安装"
  fi
  printf "\n${CYAN}── mihomo ──${NC}\n"
  if [[ -f "${SERVICE_DIR}/mihomo.service" ]]; then
    systemctl status mihomo --no-pager -l || true
  else
    warn "mihomo 未安装"
  fi
  printf "\n"
}

cmd_logs() {
  local lines="${2:-50}"
  printf "${CYAN}[proxyrelayd 日志]${NC}\n"
  journalctl -u proxyrelayd --no-pager -n "${lines}" 2>/dev/null || true
  printf "\n${CYAN}[mihomo 日志]${NC}\n"
  journalctl -u mihomo --no-pager -n "${lines}" 2>/dev/null || true
}

cmd_reset_password() {
  require_root
  log "准备重置管理员密码..."
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    err "未找到配置文件: ${CONFIG_FILE}"
    exit 1
  fi
  # 清空 password_hash 和 password，触发强制重置
  perl -0pi -e '
    s/^  password_hash:.*$/  password_hash: ""/m;
    s/^  password:.*$/  password: ""/m;
  ' "${CONFIG_FILE}"
  ok "密码已清空。正在重启服务..."
  systemctl restart proxyrelayd
  ok "重置完成！请在浏览器中使用默认账号 admin / admin 登录，并设置新密码。"
}

cmd_help() {
  cat <<EOF

ProxyRelay 服务管理工具

用法: sudo ./service.sh <命令>

可用命令:
  install     安装系统服务 (含依赖检查、构建、注册 systemd)
  uninstall   卸载 systemd 服务 (保留配置与数据)
  start       启动服务
  stop        停止服务
  restart     重启服务
  status      查看服务状态
  logs        查看最近日志
  reset-password 一键恢复默认 admin/admin 账号密码

环境变量:
  MIHOMO_BIN        mihomo 路径       (默认: /usr/local/bin/mihomo)
  MIHOMO_VERSION    mihomo 版本       (默认: v1.19.21)
  APP_DIR           项目目录          (默认: 脚本所在目录)
  CONFIG_DIR        配置目录          (默认: /etc/proxyrelay)
  STATE_DIR         数据目录          (默认: /var/lib/proxyrelay)

EOF
}

# ── entry point ───────────────────────────────────────────────────────────

case "${1:-help}" in
  install)    cmd_install ;;
  uninstall)  cmd_uninstall ;;
  start)      cmd_start ;;
  stop)       cmd_stop ;;
  restart)    cmd_restart ;;
  status)     cmd_status ;;
  logs)       cmd_logs "$@" ;;
  reset-password) cmd_reset_password ;;
  *)          cmd_help ;;
esac
