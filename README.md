# AXIS (Anycast X-Proxy Integration System)

> 基于 **Mihomo** 内核的综合代理集成系统与中转网关。将各地的远程代理节点（机场）收敛至服务器本地，统一作为中心调度的 Anycast 访问枢纽，向公网或局域网设备暴露高度稳定的 SOCKS5 / HTTP 入口。

---

## 功能概览

- **节点订阅管理** — 支持 Mihomo YAML 和 Base64 两种格式的远程订阅，可在线添加、删除、刷新与预览节点
- **出站路由群组** — 按地区正则筛选候选节点，支持 Static（手动选路）和 Dynamic（自动测速选优）两种模式
- **入站监听器** — 灵活部署 SOCKS5 / HTTP / Mixed 多协议代理端口，支持用户名密码认证
- **现代化 Web 控制台** — Vite + React 18 + Shadcn UI 构建的 SPA 后台，内置 Monaco 代码编辑器
- **安全首次使用** — 首次安装时使用默认凭据（`admin` / `admin`），登录后强制修改密码并以安全哈希保存
- **一键部署** — 提供自动化安装脚本与 systemd 服务管控，支持 Linux 环境开箱即用

---

## 架构

```
┌─────────────────────────────────────────────────┐
│              Web Console (SPA)                  │
│  系统总览 / 运行状态 / 节点订阅 / 出站路由     │
│  入站监听 / 系统配置 / 事件日志                 │
└──────────────────────┬──────────────────────────┘
                       │ HTTP API
                       ▼
┌─────────────────────────────────────────────────┐
│              AXIS Control Plane                 │
│  Node.js · 配置管理 · 状态聚合 · 运行态控制    │
└──────────────────────┬──────────────────────────┘
                       │ 渲染 mihomo.yaml
                       ▼
┌─────────────────────────────────────────────────┐
│               Mihomo (数据面)                    │
│  proxy-providers · proxy-groups · listeners     │
│  external-controller (API)                      │
└─────────────────────────────────────────────────┘
```

控制面 (`AXIS / proxyrelayd`) 负责配置、状态、订阅和 UI；数据面 (`mihomo`) 负责协议承载和流量转发。

---

## 快速开始

> **零基础一条命令启动** — `start.sh` 会自动检测并安装所有缺失的依赖（Node.js、npm、mihomo、系统工具），无需手动准备环境。

### 1. 克隆仓库

```bash
git clone https://github.com/your-org/ProxyRelay.git
cd ProxyRelay
```

### 2. 一键启动

```bash
./start.sh
```

脚本会自动完成以下全部步骤：

| 步骤 | 动作 |
|------|------|
| 环境自检 | 检测 Node.js、npm、mihomo、curl、perl、sed、ss |
| 自动安装 | 缺失的依赖将通过包管理器或 GitHub Release 自动安装 |
| 项目依赖 | 自动执行 `npm install` (后端 + 前端) |
| 前端构建 | 自动执行 `npm run build:ui` |
| 端口检查 | 检测并清理冲突的残留进程 |
| 双进程启动 | 先启动 mihomo 数据面，再启动 ProxyRelay 控制面 |

### 3. 访问控制台

浏览器打开 `http://127.0.0.1:8787`

首次登录使用默认凭据：

| 用户名  | 密码    |
|---------|---------|
| `admin` | `admin` |

> 🔒 **首次登录将强制要求修改密码**，新密码以安全哈希写入配置文件。

### 环境变量

`start.sh` 支持通过环境变量自定义行为：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MIHOMO_BIN` | `/usr/local/bin/mihomo` | mihomo 二进制路径 |
| `MIHOMO_VERSION` | `v1.19.21` | 自动安装时的 mihomo 版本 |
| `SERVER_HOST` | `0.0.0.0` | 控制面监听地址 |
| `SERVER_PORT` | `8787` | 控制面监听端口 |
| `CONTROLLER_HOST` | `127.0.0.1` | mihomo API 地址 |
| `CONTROLLER_PORT` | `11235` | mihomo API 端口 |
| `CONTROLLER_SECRET` | `proxyrelay-local-secret` | mihomo API 密钥 |

---

## 配置文件

> ⚠️ **`config/proxyrelay.yaml` 已被 `.gitignore` 排除**，不会提交到版本控制。仓库中只保留 `config/proxyrelay.example.yaml` 作为模板。首次运行 `./start.sh` 或 `sudo ./service.sh install` 时会自动从模板生成配置文件。

核心配置文件为 `config/proxyrelay.yaml`（或系统部署时为 `/etc/proxyrelay/proxyrelay.yaml`）。

```yaml
server:
  host: 0.0.0.0          # 监听地址
  port: 8787             # 控制面端口
  startup_refresh: false # 启动时是否自动刷新订阅

admin:
  username: admin
  password: ""           # 明文密码（二选一，推荐使用 hash）
  password_hash: ""      # 安全哈希（首次登录修改后自动生成）
  session_ttl_hours: 12

runtime:
  workdir: ../runtime
  mihomo_binary: mihomo                        # mihomo 路径
  external_controller: http://127.0.0.1:9090   # mihomo API
  external_secret: ""                          # mihomo Secret
  render_only: true                            # 仅渲染模式

subscriptions:
  - name: my-airport
    type: mihomo-http       # 或 clash-http（Base64）
    url: https://example.com/sub
    interval: 3600
    enabled: true
    health_check_url: https://www.gstatic.com/generate_204
    health_check_interval: 300

egress_groups:
  - name: egress-hk
    provider: my-airport
    mode: manual            # manual (Static) 或 auto (Dynamic)
    filter: "(?i)港|hk|hong ?kong"
    exclude_filter: ""

listeners:
  - name: hk-socks
    type: socks             # socks / http / mixed
    listen: 0.0.0.0
    port: 10801
    udp: true
    enabled: true
    users:
      - username: user1
        password: pass1
    egress_group: egress-hk
```

### 密码安全机制

当 `password_hash` 和 `password` 均为空时，系统启用默认凭据 `admin/admin`，首次登录后强制修改密码。修改后的密码自动以 Scrypt 哈希写入 `password_hash` 字段。

生成密码哈希的命令行方式：

```bash
npm run hash-password -- <your-password>
```

### 开源分发

开源时只需确保 `admin.password_hash` 和 `admin.password` 均为空字符串，用户首次安装后会经历：

1. 使用 `admin` / `admin` 登录
2. 系统强制弹出密码修改界面
3. 新密码自动回写到配置文件

---

## API 一览

所有 API 以 `/api/` 前缀访问，需先通过 `POST /api/session` 获取会话。

| 方法   | 路径                              | 说明                     |
|--------|-----------------------------------|--------------------------|
| POST   | `/api/session`                    | 登录                     |
| GET    | `/api/session`                    | 查询会话                 |
| DELETE | `/api/session`                    | 注销                     |
| PUT    | `/api/session/password`           | 修改密码                 |
| GET    | `/api/status`                     | 系统总览                 |
| GET    | `/api/config`                     | 读取配置                 |
| PUT    | `/api/config`                     | 保存配置                 |
| GET    | `/api/rendered-config`            | 获取渲染后的 mihomo.yaml |
| GET    | `/api/providers`                  | 订阅列表                 |
| POST   | `/api/providers/:name/refresh`    | 刷新指定订阅             |
| GET    | `/api/groups`                     | 出站路由组列表           |
| POST   | `/api/groups/:name/select`        | 切换节点                 |
| POST   | `/api/groups/:name/healthcheck`   | 触发健康检查             |
| GET    | `/api/listeners`                  | 监听器列表               |
| GET    | `/api/events`                     | 事件日志                 |
| GET    | `/api/controller`                 | Mihomo 控制器状态        |
| POST   | `/api/controller/probe`           | 探测 Mihomo 连通性       |
| GET    | `/api/runtime-preflight`          | 运行预检                 |
| POST   | `/api/reload`                     | 重载运行态               |
| POST   | `/api/subscription-test`          | 临时测试订阅             |

---

## 运行模式

### render-only（默认）

仅做配置渲染，不要求 Mihomo 运行。适合：

- 前端开发调试
- 配置预审

在 `proxyrelay.yaml` 中设置：

```yaml
runtime:
  render_only: true
```

### managed

需要 Mihomo 运行并可通过 external-controller 通信。适合：

- 生产环境
- 联调测试

```yaml
runtime:
  render_only: false
  external_controller: http://127.0.0.1:9090
  external_secret: "your-secret"
```

---

## 系统服务部署 (systemd)

项目自带 `service.sh` 脚本，可将 AXIS 注册为系统服务并管理其生命周期。

### 一键安装为系统服务

```bash
sudo ./service.sh install
```

`install` 命令会自动完成：

- ✅ 检测并安装 Node.js、mihomo 等依赖
- ✅ 创建 `proxyrelay` 系统用户与安全目录
- ✅ 生成生产配置文件 `/etc/proxyrelay/proxyrelay.yaml`
- ✅ 安装后端依赖并构建前端
- ✅ 渲染初始运行态配置
- ✅ 注册 `proxyrelayd` 与 `mihomo` 两个 systemd 服务

### 服务管理命令

```bash
sudo ./service.sh start      # 启动服务
sudo ./service.sh stop       # 停止服务
sudo ./service.sh restart    # 重启服务
sudo ./service.sh status     # 查看运行状态
sudo ./service.sh logs       # 查看最近日志
sudo ./service.sh uninstall  # 卸载 systemd 服务 (保留配置与数据)
```

### 推荐目录结构

```
/opt/proxyrelay/                  # 应用代码 (APP_DIR)
/etc/proxyrelay/proxyrelay.yaml   # 配置文件 (CONFIG_DIR)
/var/lib/proxyrelay/runtime/      # 运行时产物 (STATE_DIR)
```

### 环境变量

`service.sh` 支持通过环境变量自定义安装路径：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_DIR` | 脚本所在目录 | 应用代码目录 |
| `CONFIG_DIR` | `/etc/proxyrelay` | 配置文件目录 |
| `STATE_DIR` | `/var/lib/proxyrelay` | 运行数据目录 |
| `MIHOMO_BIN` | `/usr/local/bin/mihomo` | mihomo 路径 |
| `MIHOMO_VERSION` | `v1.19.21` | 自动安装 mihomo 版本 |
| `PROXYRELAY_USER` | `proxyrelay` | 运行用户 |

### 运行预检

```bash
node src/index.js preflight
node src/index.js preflight --json
```

---

## 防火墙建议

| 端口         | 用途                 | 建议          |
|-------------|----------------------|---------------|
| `8787`      | AXIS 控制面与交互台    | 仅内网访问    |
| `9090`      | Mihomo Controller    | 仅本机访问    |
| `10801-...` | SOCKS5/HTTP 代理入口 | 按需公网开放  |

---

## 项目结构

```
ProxyRelay/
├── config/
│   └── proxyrelay.example.yaml  # 配置模板 (首次运行自动生成 proxyrelay.yaml)
├── deploy/
│   ├── install-ubuntu.sh     # Ubuntu 安装脚本 (legacy)
│   ├── start-local.sh        # 本地联调启动脚本 (被 start.sh 调用)
│   ├── proxyrelayd.service   # systemd 控制面服务模板
│   └── mihomo.service        # systemd 数据面服务模板
├── docs/                     # 补充文档
├── frontend/                 # React SPA 前端
│   └── src/
│       ├── pages/            # 页面组件
│       ├── layouts/          # 布局组件
│       └── components/       # UI 组件库
├── runtime/                  # 运行时产物（自动生成）
├── src/
│   ├── index.js              # 入口与 CLI
│   ├── server.js             # HTTP 服务器
│   ├── lib/
│   │   ├── auth.js           # 认证与哈希
│   │   ├── config.js         # 配置读写
│   │   └── pattern.js        # 正则编译
│   └── services/
│       ├── control-plane.js  # 核心控制面逻辑
│       ├── mihomo-controller.js
│       ├── mihomo-renderer.js
│       ├── runtime-preflight.js
│       └── subscription-service.js
├── test/                     # 测试用例
├── package.json
├── start.sh                  # ⚡ 一键启动 (自动安装依赖 + 前台运行)
├── service.sh                # 🔧 系统服务管理 (install/start/stop/restart)
└── README.md
```

---

## 常用命令

```bash
# ⚡ 一键启动 (自动检测 + 安装依赖 + 启动双进程)
./start.sh

# 🔧 注册为系统服务
sudo ./service.sh install
sudo ./service.sh start
sudo ./service.sh stop
sudo ./service.sh restart
sudo ./service.sh status
sudo ./service.sh logs

# 仅启动控制面 (render-only，无需 mihomo)
npm start

# 开发热更
npm run dev

# 构建前端
npm run build:ui

# 运行测试
npm test

# 生成密码哈希
npm run hash-password -- <password>

# 运行预检
npm run preflight
```

---

## 技术栈

| 层       | 技术                                            |
|----------|------------------------------------------------|
| 控制面   | Node.js (原生 HTTP, ESM)                        |
| 数据面   | Mihomo                                          |
| 前端     | React 18 + Vite + Shadcn UI + Lucide Icons     |
| 编辑器   | Monaco Editor                                   |
| 认证     | Scrypt 密码哈希 + Cookie Session                |
| 配置     | YAML (proxyrelay.yaml → mihomo.yaml 渲染)      |
| 部署     | systemd                                         |

---

## License

MIT
