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
| 本地配置 | 生成或复用开发 profile：`~/.../AXIS/dev/config/proxyrelay.yaml` |
| 后端启动 | 启动 AXIS API，并监听 `8787` |
| 前端启动 | 启动 React + Vite 开发服务器，并监听 `5173` |
| 联调代理 | Vite 自动把 `/api/*` 代理到后端 |
| 页面访问 | Go 服务会把非 `/api/*` 请求反向代理到 Vite，保留热更新体验 |

### 4. 打开控制台

浏览器访问：

```text
http://127.0.0.1:8787
```

如果你想直接访问 Vite 开发服务器，也可以打开：

```text
http://127.0.0.1:5173
```

现在默认入口会先进入 `Launch` 页面，由它判断下一步：

- 首次初始化未完成：跳去 `/setup`
- 未登录且初始化已具备管理员账号：跳去 `/login`
- 已登录：直接进 `/dashboard`
- 已登录且初始化完成：跳去 `/dashboard`

`reset:setup` 之后不会再要求你先用默认账号登录。
你会先进入 `Setup` 页面，自己输入管理员账号和密码完成初始化，随后前端会自动登录并直接进入控制台。
如果订阅、出口线路、本地入口还没补齐，控制台里会继续提示，但不会再强制卡在 `Setup`。

### 5. 开发环境默认就是完整能力

默认开发模式会直接让 AXIS 接管运行时，不是“只写配置不下发”的裁剪版。

如果你只是想调界面或调配置，不希望当前开发环境真的接管运行时，可以显式切到仅渲染模式：

```bash
pnpm dev:web:render-only
```

如果你想把当前开发环境重置回“重新走 Setup”的状态，可以执行：

```bash
pnpm reset:setup
```

这个命令会：

- 把开发 profile 的配置重置回未初始化状态
- 清空开发 profile 下的运行时目录、缓存、状态库和开发期痕迹
- 失效当前开发 profile 下已有会话
- 下次启动时重新从 Launch -> Login / Setup 流程开始

如果桌面端当前已经在运行，重置后还需要先彻底退出现有桌面进程，再重新打开，才能读到新的初始化状态。

## 仓库怎么管理

这个项目不是“主仓库 + 一个前端子项目单独管理”的方式，而是标准 repo/workspace 结构：

- 根目录使用 `pnpm workspace` 管理 Node 依赖和脚本
- 日常开发不需要进入 `frontend/`
- 常用命令都在根目录执行

最常用的是这几个：

```bash
pnpm dev
pnpm dev:desktop
pnpm dev:web:render-only
pnpm dev:desktop:render-only
pnpm build
pnpm build:desktop
pnpm lint
pnpm test
```

其中：

- `pnpm dev` 保留 Vite 热更新，并默认启用 AXIS 的完整运行时能力
- `pnpm dev:desktop` 启动 Wails v2 桌面壳，保留前端热更新，并默认启用 AXIS 的完整运行时能力
- `pnpm build` 会先构建前端，再把 `frontend/dist` 内嵌进 Go 二进制
- `pnpm build:desktop` 会先生成最新 OpenAPI 类型，再走 Wails 桌面打包流程

## 数据目录与路径规则

AXIS 现在统一采用和 Electron 类似的 `userData` 目录策略。

这套规则只有两层：

- 正式运行数据：`userData/AXIS`
- 开发 profile 数据：`userData/AXIS/dev`

默认情况下：

- CLI `go run ./cmd/axis serve`
- 桌面应用直接启动
- `pnpm dev`
- `pnpm dev:desktop`
- `pnpm reset:setup`

都会围绕这套目录工作，不再默认往仓库根目录写运行时文件，也不再根据 `cwd` 猜测其他位置。

### 默认目录

正式运行：

```text
macOS:   ~/Library/Application Support/AXIS
Windows: %APPDATA%/AXIS
Linux:   ~/.config/AXIS
```

开发 profile：

```text
macOS:   ~/Library/Application Support/AXIS/dev
Windows: %APPDATA%/AXIS/dev
Linux:   ~/.config/AXIS/dev
```

### 目录结构

正式运行根目录和开发 profile 根目录内部结构保持一致：

```text
AXIS_HOME/
  config/
    proxyrelay.yaml
  runtime/
    axis.db
    control-state.json
    mihomo.yaml
    mihomo.last-good.yaml
    providers/
    mihomo/
      versions/
  updater/
    state.json
    command.json
    updater.log
  logs/
```

### 覆盖规则

只有两种显式覆盖方式：

- `AXIS_HOME`
  - 覆盖整个数据根目录
- `PROXYRELAY_CONFIG`
  - 只覆盖配置文件路径

如果你没有显式设置这两个变量，AXIS 永远只会落到默认 `userData/AXIS` 或 `userData/AXIS/dev`，不会偷偷回退到仓库目录、应用同级目录或别的用户目录。

### 构建产物和运行态产物的区别

这两个不要混淆：

- 构建产物仍然留在仓库里
  - `./desktop/build`
  - `./.tmp/desktop`
- 运行态产物统一去 `userData`
  - 配置
  - 数据库
  - 渲染结果
  - updater 状态
  - 开发 profile 痕迹

## 桌面模式

如果你要直接开发桌面壳，而不是浏览器里的 Web 控制台：

```bash
pnpm dev:desktop
```

如果你只想调界面和配置，不希望桌面开发环境真的接管运行时：

```bash
pnpm dev:desktop:render-only
```

如果你只想让 Web 开发环境停留在“仅渲染配置”：

```bash
pnpm dev:web:render-only
```

桌面模式的实现方式是：

- Go 核心服务保持不变
- Wails v2 直接绑定同一套 Go Service
- React 前端通过 `api` provider 自动判断当前走 HTTP 还是 Wails 原生调用
- 启动入口统一先走 Launch 状态页
- 桌面模式会额外拉起一个独立 updater 守护进程，用于检查 GitHub Releases

桌面应用启动时的配置来源是：

1. 如果设置了 `PROXYRELAY_CONFIG`，优先使用它
2. 否则固定使用 `userData/AXIS/config/proxyrelay.yaml`
3. 如果配置不存在，会在该目录首启生成默认配置

开发脚本不会再往仓库根目录写运行时产物，而是固定落到开发 profile：

```text
macOS:   ~/Library/Application Support/AXIS/dev
Windows: %APPDATA%/AXIS/dev
Linux:   ~/.config/AXIS/dev
```

正式桌面/CLI 默认配置位置是：

```text
macOS:   ~/Library/Application Support/AXIS/config/proxyrelay.yaml
Windows: %APPDATA%/AXIS/config/proxyrelay.yaml
Linux:   ~/.config/AXIS/config/proxyrelay.yaml
```

如果要构建桌面产物：

```bash
pnpm build:desktop
```

可选地，你也可以通过环境变量指定 Wails 目标平台：

```bash
AXIS_WAILS_PLATFORM=windows/amd64 pnpm build:desktop
```

在 macOS 上，`build:desktop` 现在会自动做桌面产物后处理：

- 递归清理产物上的扩展属性
- 默认执行一次 ad-hoc 签名，避免产物还是“完全裸包”
- 如果你提供 `Developer ID` 证书，还会改用正式签名
- 如果你再提供 `notarytool` profile，还会继续做 notarization 和 staple

示例：

```bash
AXIS_MAC_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)" pnpm build:desktop
```

```bash
AXIS_MAC_SIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)" \
AXIS_MAC_NOTARY_PROFILE="axis-notary" \
pnpm build:desktop
```

说明：

- 没有 `AXIS_MAC_SIGN_IDENTITY` 时，脚本只会做 ad-hoc 签名，适合本机自测
- 要发给别的机器长期使用，建议一定配 `Developer ID` 签名
- 需要避免 Gatekeeper / “无法验证开发者” 一类问题时，还应继续做 notarization
- `xattr` 只能处理下载隔离标记，不能替代代码签名和 notarization
- 当前机器如果没有可用 codesigning identity，产物仍然只是“本机测试包”

## 配置文件

`config/proxyrelay.yaml` 已被 `.gitignore` 排除，不会提交到版本控制。仓库中保留的是 `config/proxyrelay.example.yaml` 模板。

如果你直接运行 AXIS CLI 或桌面应用，而没有显式设置 `PROXYRELAY_CONFIG`，默认配置文件路径是：

```text
macOS:   ~/Library/Application Support/AXIS/config/proxyrelay.yaml
Windows: %APPDATA%/AXIS/config/proxyrelay.yaml
Linux:   ~/.config/AXIS/config/proxyrelay.yaml
```

如果你通过 `service.sh` 或 systemd 部署，服务模板会显式设置自己的配置文件路径，例如：

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

当 `password_hash` 和 `password` 都为空时，系统会把当前 profile 视为“未初始化”。
这时 `Launch` 会优先跳去 `Setup`，要求你先创建管理员账号和密码；创建完成后，新密码会写入 `password_hash`，旧的默认登录入口不会再放行。

如果你想自己先生成密码哈希：

```bash
pnpm run hash-password -- <your-password>
```

## API 一览

AXIS 现在有两类 API：

- 公共动态代理 API：给外部程序直接取代理，不需要登录
- 启动入口 API：给 Launch 页和首次初始化流程判断下一步，不要求已登录
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

### 启动入口 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/bootstrap/status` | 返回 Launch 页所需的宿主、主服务、更新服务、登录态和 Setup 状态 |
| GET | `/api/setup-state` | 返回首次初始化状态，未登录也可访问 |
| PUT | `/api/setup/admin` | 在首次初始化阶段创建管理员账号和密码 |

### 控制面 API

所有控制面 API 都以 `/api/` 开头，先通过 `POST /api/session` 登录。
如果当前还没完成首次初始化，`POST /api/session` 会直接返回错误，并要求先走 `Setup`。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/session` | 登录 |
| GET | `/api/session` | 查询会话 |
| DELETE | `/api/session` | 注销 |
| PUT | `/api/session/password` | 修改密码 |
| GET | `/api/status` | 系统总览 |
| GET | `/api/host/status` | 查询宿主状态 |
| GET | `/api/host/updater` | 查询桌面 updater 状态 |
| POST | `/api/host/updater/check` | 触发桌面 updater 立即检查版本 |
| POST | `/api/host/open-browser` | 打开桌面控制台 |
| PUT | `/api/host/autostart` | 设置桌面宿主开机自启 |
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

## 运行开关

### 仅渲染配置

适合：

- 前端开发
- 配置调试
- 先把线路、入口和文案整理好

```yaml
runtime:
  render_only: true
```

### 接管运行时

这是 AXIS 的正常工作状态。AXIS 会通过 `external-controller` 接管 Mihomo，并在需要时下发配置。

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
│   └── proxyrelay.example.yaml  # 仓库只保留模板，不再作为默认运行态配置位置
├── desktop/                  # Wails v2 桌面壳入口与打包配置
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
├── scripts/
│   ├── axis-paths.mjs        # AXIS userData / dev profile 路径计算
│   ├── dev.mjs               # Web 开发入口
│   ├── desktop-dev.mjs       # 桌面开发入口
│   ├── desktop-build.mjs     # 桌面构建与签名后处理
│   └── reset-setup.mjs       # 重置开发 profile，重新走 Setup
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
pnpm dev

# 开发数据固定写到 userData/AXIS/dev
# 入口会先进入 Launch，再自动分流到 Login / Setup / Dashboard

# 启动桌面壳（保留前端热更新，默认接管运行时）
pnpm dev:desktop

# 只渲染配置，不接管运行时
pnpm dev:web:render-only
pnpm dev:desktop:render-only

# 重置开发 profile，重新走 Setup
pnpm reset:setup

# 构建
pnpm build

# 构建桌面产物
pnpm build:desktop

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
