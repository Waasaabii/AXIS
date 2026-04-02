# AXIS

> 一个给人用的代理控制台。
>
> 你把机场订阅导入 AXIS，把原始节点整理成几条好理解的线路，再给浏览器、系统或其他设备开本地入口。需要固定最终出口时，还可以在线路后面挂一个落地节点。

## 它适合拿来做什么

AXIS 主要解决这几件事：

- 把多个机场订阅放到一个地方统一管理
- 不再直接面对一大堆原始节点，而是整理成“香港日常”“日本流媒体”“备用线路”这种可读名字
- 给浏览器、操作系统、电视盒子或其他设备开一个稳定的本地代理入口
- 让不同线路最终落到不同公网 IP
- 用一个网页控制台完成查看、修改、切换和排障

如果你想把它理解成一句话：

> AXIS 想做的是一个更容易上手、也更适合长期管理的 Mihomo 控制台。

## AXIS 现在的结构

AXIS 是单仓库结构：

- 后端：Go 原生服务，负责配置、状态、API、Mihomo 配置渲染
- 前端：React 控制台，负责用户操作体验
- 数据面：Mihomo，负责真正的流量转发

也就是说：

1. React 负责“让你配得明白”
2. Go 负责“把你的配置组织正确”
3. Mihomo 负责“真的把流量转出去”

### 架构图

```mermaid
flowchart LR
  UI["React 控制台"] --> API["AXIS 控制面<br/>Go 原生服务"]
  API --> CFG["生成 mihomo.yaml"]
  CFG --> CORE["Mihomo 数据面"]
  CORE --> NET["公网 / 目标服务"]
```

## 一条流量是怎么走的

AXIS 里最重要的是这三个概念：

- `订阅与节点`：机场原始节点来源
- `出口线路`：把原始节点整理成一条可复用路径
- `本地入口`：设备真正连接的地址和端口

如果你还配置了落地节点，实际链路会是这样：

```mermaid
flowchart LR
  DEVICE["你的设备<br/>浏览器 / 系统 / 电视"] --> LISTENER["本地入口<br/>SOCKS5 / HTTP / Mixed"]
  LISTENER --> GROUP["出口线路"]
  GROUP --> NODE["当前选中的订阅节点"]
  NODE --> LANDING["落地节点（可选）"]
  LANDING --> INTERNET["公网"]
```

没有落地节点时，链路会简化为：

```text
设备 -> 本地入口 -> 出口线路 -> 当前选中的订阅节点 -> 公网
```

## 快速开始

> 默认开发入口是跨平台的 `pnpm dev`。不需要切进 `frontend/` 单独管理子项目，所有命令都在仓库根目录执行。

### 1. 克隆仓库

```bash
git clone https://github.com/Waasaabii/AXIS.git
cd AXIS
```

### 2. 安装依赖

```bash
pnpm install
```

### 3. 启动开发环境

```bash
pnpm dev
```

这个命令会同时做几件事：

| 步骤 | 动作 |
|------|------|
| 本地配置 | 基于 `config/proxyrelay.yaml` 或模板生成 `runtime/dev/proxyrelay.dev.yaml` |
| 后端启动 | 启动 AXIS API，并监听 `8787` |
| 前端启动 | 启动 React + Vite 开发服务器，并监听 `5173` |
| 联调代理 | Vite 自动把 `/api/*` 代理到后端 |

### 4. 打开控制台

浏览器访问：

```text
http://127.0.0.1:5173
```

如果你使用的是开源模板配置，首次登录默认凭据是：

| 用户名 | 密码 |
|--------|------|
| `admin` | `admin` |

登录后会被要求修改密码，新的密码会自动写入安全哈希。

### 5. 如果你想连本地 Mihomo

默认开发模式是 `render-only`，只渲染配置，不真的接管运行中的 Mihomo。

如果你想一起联调本地 Mihomo，可以用：

```bash
pnpm dev:managed
```

## 仓库怎么管理

这个项目不是“主仓库 + 一个前端子项目单独管理”的方式，而是标准 repo/workspace 结构：

- 根目录使用 `pnpm workspace` 管理 Node 依赖和脚本
- 日常开发不需要进入 `frontend/`
- 常用命令都在根目录执行

最常用的是这几个：

```bash
pnpm dev
pnpm dev:managed
pnpm build
pnpm lint
pnpm test
```

## 配置文件

`config/proxyrelay.yaml` 已被 `.gitignore` 排除，不会提交到版本控制。仓库中保留的是 `config/proxyrelay.example.yaml` 模板。

部署到系统服务时，默认配置文件路径是：

```text
/etc/proxyrelay/proxyrelay.yaml
```

### 推荐理解顺序

建议按这个顺序理解配置：

1. 先加 `subscriptions`
2. 再决定是否需要 `landing_proxies`
3. 然后把节点整理成 `egress_groups`
4. 最后创建设备要连接的 `listeners`

### 配置示例

```yaml
server:
  host: 0.0.0.0
  port: 8787
  startup_refresh: false

admin:
  username: admin
  password: ""
  password_hash: ""
  session_ttl_hours: 12

runtime:
  workdir: ../runtime
  mihomo_binary: mihomo
  external_controller: http://127.0.0.1:9090
  external_secret: ""
  render_only: true

subscriptions:
  - name: airport-main
    type: mihomo-http
    url: https://example.com/sub
    interval: 3600
    enabled: true
    health_check_url: https://www.gstatic.com/generate_204
    health_check_interval: 300

landing_proxies:
  - name: tokyo-exit
    type: socks5
    server: landing.example.com
    port: 443
    username: relay-user
    password: relay-pass
    enabled: true

egress_groups:
  - name: hk-daily
    provider: airport-main
    mode: manual
    filter: "(?i)港|hk|hong ?kong"
    exclude_filter: ""

  - name: us-auto
    provider: airport-main
    mode: auto
    filter: "(?i)美|us|united states|america"
    exclude_filter: ""
    landing_proxy: tokyo-exit

listeners:
  - name: browser-default
    type: mixed
    listen: 0.0.0.0
    port: 10801
    udp: true
    enabled: true
    users:
      - username: user1
        password: change-me
    egress_group: us-auto
```

### 这份配置到底表示什么

- `airport-main` 是订阅来源
- `tokyo-exit` 是固定的最终出口
- `us-auto` 会从订阅里挑出美国节点，再把流量转到 `tokyo-exit`
- `browser-default` 是你的设备真正要连接的代理地址和端口

对应的实际链路是：

```text
浏览器 -> browser-default -> us-auto -> 订阅中的美国节点 -> tokyo-exit -> 公网
```

### 密码安全机制

当 `password_hash` 和 `password` 都为空时，系统允许你用 `admin/admin` 首次登录，并强制修改密码。修改后的密码会自动写入 `password_hash`。

如果你想自己先生成密码哈希：

```bash
pnpm run hash-password -- <your-password>
```

## API 一览

AXIS 现在有两类 API：

- 公共动态代理 API：给外部程序直接取代理，不需要登录
- 控制面 API：给控制台和运维使用，需要先登录

### 公共动态代理 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 查看动态代理服务健康状态 |
| GET | `/api/proxy/next` | 获取下一个可用代理，默认返回 `text/plain` |

`/api/proxy/next` 支持这些可选查询参数：

| 参数 | 说明 |
|------|------|
| `protocol` | 期望协议，支持 `http` 或 `socks5` |
| `region` | 地区筛选 |
| `city` | 城市筛选 |
| `tag` | 标签筛选 |
| `session` | 粘性会话标识 |
| `format` | 返回格式，支持 `text` 或 `json` |

默认返回示例：

```text
socks5://username:password@proxy.example.com:10801
```

如果你传 `?format=json`，会返回带元信息的 JSON。

### 控制面 API

所有控制面 API 都以 `/api/` 开头，先通过 `POST /api/session` 登录。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/session` | 登录 |
| GET | `/api/session` | 查询会话 |
| DELETE | `/api/session` | 注销 |
| PUT | `/api/session/password` | 修改密码 |
| GET | `/api/status` | 系统总览 |
| GET | `/api/config` | 读取完整配置 |
| PUT | `/api/config` | 保存完整配置 |
| GET | `/api/providers` | 订阅与节点列表 |
| POST | `/api/providers/:name/refresh` | 刷新订阅 |
| GET | `/api/groups` | 出口线路列表 |
| POST | `/api/groups/:name/select` | 切换当前节点 |
| POST | `/api/groups/:name/healthcheck` | 重新检测线路节点 |
| GET | `/api/landing-proxies` | 落地节点列表 |
| POST | `/api/landing-proxies` | 创建落地节点 |
| PUT | `/api/landing-proxies/:name/update` | 更新落地节点 |
| DELETE | `/api/landing-proxies/:name` | 删除落地节点 |
| GET | `/api/listeners` | 本地入口列表 |
| POST | `/api/listeners` | 创建本地入口 |
| PUT | `/api/listeners/:name/update` | 更新本地入口 |
| DELETE | `/api/listeners/:name` | 删除本地入口 |
| GET | `/api/rendered-config` | 查看渲染后的 `mihomo.yaml` |
| GET | `/api/events` | 事件日志 |
| GET | `/api/controller` | Mihomo 控制器状态 |
| POST | `/api/controller/probe` | 重新探测连通性 |
| GET | `/api/runtime-preflight` | 运行前检查 |
| POST | `/api/reload` | 重载运行态 |
| POST | `/api/subscription-test` | 临时测试订阅 |

## 运行模式

### render-only

默认模式。只渲染配置，不要求 Mihomo 已经运行。

适合：

- 前端开发
- 配置调试
- 先把线路、入口和文案整理好

```yaml
runtime:
  render_only: true
```

### managed

AXIS 会尝试通过 `external-controller` 接管 Mihomo。

适合：

- 本地联调
- 生产部署
- 需要一键刷新和下发配置

```yaml
runtime:
  render_only: false
  external_controller: http://127.0.0.1:9090
  external_secret: "your-secret"
```

## systemd 部署

项目自带 `service.sh`，可以把 AXIS 安装为 systemd 服务。

### 安装

```bash
sudo ./service.sh install
```

这个命令会完成：

- 准备运行目录
- 生成生产配置
- 安装前端构建产物
- 渲染初始运行态配置
- 注册 `proxyrelayd` 和 `mihomo` 两个 systemd 服务

### 常用管理命令

```bash
sudo ./service.sh start
sudo ./service.sh stop
sudo ./service.sh restart
sudo ./service.sh status
sudo ./service.sh logs
sudo ./service.sh reset-password
sudo ./service.sh uninstall
```

## 防火墙建议

| 端口 | 用途 | 建议 |
|------|------|------|
| `8787` | AXIS 控制台和 API | 仅内网访问 |
| `9090` | Mihomo Controller | 仅本机访问 |
| `10801-...` | 代理入口 | 按需开放 |

## 项目结构

```text
AXIS/
├── cmd/
│   └── axis/                 # Go CLI 入口（serve / preflight / openapi / hash-password）
├── config/
│   └── proxyrelay.example.yaml
├── deploy/
│   ├── install-ubuntu.sh
│   ├── proxyrelayd.service
│   └── mihomo.service
├── frontend/
│   └── src/
│       ├── pages/            # 页面
│       ├── layouts/          # 布局
│       ├── components/       # UI 组件
│       ├── services/         # API 封装
│       └── generated/        # OpenAPI 生成类型
├── internal/
│   └── axis/                 # Go 后端核心实现
├── runtime/                  # 运行时产物（自动生成）
├── scripts/
│   └── dev.mjs               # 本地开发入口
├── service.sh                # systemd 服务管理脚本
├── package.json
├── pnpm-workspace.yaml
└── README.md
```

## 常用命令

```bash
# 安装依赖
pnpm install

# 启动前端 + 后端
pnpm dev

# 启动前端 + 后端 + 本地 Mihomo
pnpm dev:managed

# 构建
pnpm build

# 代码检查
pnpm lint

# 测试
pnpm test

# 单独启动后端 / 前端
pnpm run dev:server
pnpm run dev:ui

# 生成密码哈希
pnpm run hash-password -- <password>

# 运行预检
pnpm run preflight
```

## 技术栈

| 层 | 技术 |
|----|------|
| 控制面 | Go 原生 HTTP 服务 |
| 前端 | React 19 + Vite + Shadcn UI |
| 数据面 | Mihomo |
| 认证 | Scrypt 密码哈希 + Cookie Session |
| 配置 | YAML |
| 部署 | systemd |

## License

MIT
