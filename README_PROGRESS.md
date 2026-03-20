# ProxyRelay 进度记录

## 当前状态

日期：2026-03-20

当前项目已经完成三条主线的收口：

- **V0.5 Ubuntu 联调主线**：已具备真实 `mihomo` managed 运行态接入能力，`controller`、`/version`、`/configs`、运行态下发链路均已实机验证。
- **后台 UI 现代化主线 (SPA 重构)**：已摒弃老旧的多页面 HTML 架构，基于 **Vite + React 18 + Tailwind CSS v4 + Shadcn UI** 完全重写为精美的现代企业级单页应用 (SPA)。包含完整的路由、全局拦截和组件化状态管理。
- **构建与部署闭环**：通过集成 `build:ui` 和重写服务端路由，实现了“本地开发支持前端热更，生产部署自动打包，后端一键 Serve” 的极简运维体验。

项目已经不再停留在“纯骨架”或“仅渲染配置”的阶段，而是进入：

- 可本地以 render-only 或本地 managed 模式启动
- 可在 Ubuntu 上以 systemd 或手工双进程方式运行
- 可通过现代化控制面 UI 对运行中的 `mihomo` 发起状态查询与 reload
- 可在极致流畅的交互中管理配置（内置 Monaco Editor）

## 本轮同步重点

本轮在原有 `V0.5 Ubuntu 联调支撑版` 的基础上，完成了全面的前端重塑：

- 彻底移除了原 `public/` 目录下的老旧多页面结构。
- 引入完整的 React 生态：`react-router-dom`、`swr`、`lucide-react` 等。
- 引入 `Shadcn UI` 实现了统一、极简、高质的 Zinc 主题。
- 将后端 `src/server.js` 改造为支持 SPA 路由回退 (Fallback) 的混合服务器。
- 在 `install-ubuntu.sh` 及 `start-local.sh` 脚本中嵌入了前端自动构建逻辑 (`npm run build:ui`)。
- 集成了 `@monaco-editor/react`，大幅提升了“系统配置”页面的使用体验。


## 已实现内容

### 控制面 API

- `POST /api/session`
- `GET /api/session`
- `DELETE /api/session`
- `GET /api/status`
- `GET /api/config`
- `PUT /api/config`
- `GET /api/providers`
- `GET /api/groups`
- `GET /api/listeners`
- `GET /api/events`
- `GET /api/rendered-config`
- `GET /api/controller`
- `GET /api/runtime-preflight`
- `POST /api/reload`
- `POST /api/subscription-test`
- `POST /api/controller/probe`
- `POST /api/providers/:name/refresh`
- `POST /api/groups/:name/select`
- `POST /api/groups/:name/healthcheck`


### 配置、渲染与运行态

- 读取 `config/proxyrelay.yaml`
- 读取 `/etc/proxyrelay/proxyrelay.yaml`
- 校验订阅、出口组、监听端口配置
- 渲染 `runtime/mihomo.yaml`
- 保留 `runtime/mihomo.last-good.yaml`
- 保存 `runtime/control-state.json`
- 保存配置后尝试把最新 `mihomo.yaml` 下发到 running controller
- 重载配置后尝试把最新 `mihomo.yaml` 下发到 running controller
- 在 runtime 状态中记录：
  - `lastRenderAt`
  - `lastApplyAt`
  - `lastApplyStatus`
  - `lastApplyMessage`
- 根据配置文件路径自动推导默认 runtime 目录：
  - 仓库内配置默认走 `../runtime`
  - `/etc/proxyrelay/proxyrelay.yaml` 默认走 `/var/lib/proxyrelay/runtime`


### 运行预检

- 提供 `node src/index.js preflight`
- 支持 `node src/index.js preflight --json`
- 提供 `GET /api/runtime-preflight`
- 检查项已覆盖：
  - runtime 目录是否与 Ubuntu 目标路径对齐
  - runtime / providers 是否可写
  - `mihomo.yaml` / `mihomo.last-good.yaml` / `control-state.json` 是否存在
  - `mihomo` 二进制是否检测到
  - controller secret 是否配置
  - controller 是否可达
  - 最近一次运行态下发状态
  - systemd service 文件是否就位
- 输出中已包含：
  - `status`
  - `ready`
  - `summary`
  - `paths`
  - `checks`
  - `recommendations`


### 订阅预览

已支持解析：

- VMess
- VLESS
- Shadowsocks
- Clash YAML 中的 `proxies`


### 前端后台

当前后台已彻底重构为基于 **Vite + React + Tailwind + Shadcn UI** 的现代化单页应用 (SPA)。

核心路由结构：

- `/dashboard`：总览面板
- `/status`：状态与预检
- `/subscriptions`：订阅与出口组配置
- `/interfaces`：对外接口配置
- `/system`：系统配置 (内建 Monaco Editor)
- `/events`：事件日志

对应能力包括：

- 现代化登录页与 Auth 拦截
- SWR 实现的无刷新数据同步与状态聚合
- Shadcn 提供的全局统一化极简 UI (Card, Form, Dropdown, Table)
- Sonner 提供的优雅全局 Toast 提示系统
- App Shell 布局 (左侧 Sidebar + 顶部快捷栏)
- Monaco Editor 强化的代码编辑体验
- 预检状态图标可视化分级

### 本地与部署资产

- `deploy/proxyrelayd.service`
- `deploy/mihomo.service`
- `deploy/install-ubuntu.sh`
- `deploy/start-local.sh`
- `docs/deployment.md`
- `docs/mihomo-runtime.md`


### 测试

- `auth.test.js`
- `config.test.js`
- `mihomo-controller.test.js`
- `mihomo-renderer.test.js`
- `runtime-preflight.test.js`
- `subscription-service.test.js`


## 已验证

### 代码与接口

- `npm test` 通过
- 当前测试数为 `11/11`
- 本地服务可启动
- 登录接口可用
- Cookie 会话可用
- `GET /api/status` 可返回总览
- `runtime/mihomo.yaml` 可生成
- `GET /api/config` / `PUT /api/config` 可读写配置
- `POST /api/subscription-test` 可完成临时订阅测试
- `GET /api/rendered-config` 可返回渲染结果
- `GET /api/controller` / `POST /api/controller/probe` 可返回 adapter 状态
- managed 模式下可通过 controller 尝试下发最新 `mihomo.yaml`
- `node src/index.js preflight` 可输出 Ubuntu 运行预检结果
- `node src/index.js preflight --json` 可输出结构化 JSON
- `GET /api/runtime-preflight` 可返回预检结果

### Ubuntu 实机链路

- 已在当前 Ubuntu 主机安装 `mihomo`
- 已完成 `/opt/proxyrelay`、`/etc/proxyrelay`、`/var/lib/proxyrelay` 初始化
- 已验证 `deploy/install-ubuntu.sh` 可完成系统目录和 service 模板安装
- 已验证 `proxyrelayd` 和 `mihomo` 可通过 systemd 启动
- 已验证 `controller` 可达
- 已验证 `POST /api/reload` 可触发运行态下发
- 已验证 `mihomo /version`
- 已验证 `mihomo /configs`
- 已验证 `preflight` 在已接入环境下可达成全绿

### 前端后台

- 现代化 React SPA 入口及路由处理：
  - `/dashboard`
  - `/status`
  - `/subscriptions`
  - `/interfaces`
  - `/system`
  - `/events`
- `src/server.js` 现已支持前端 Vite 构建产物 Serve 及 SPA 的 history fallback (`index.html` 路由退回)。
- 新版灰白样式 (Tailwind/Shadcn) 与应用外壳已通过本地及 MCP 实机验证。


## 当前阻塞

当前剩余的真实阻塞已经从“运行态能不能起来”转为“业务链路有没有接真实数据”：

- 示例订阅 URL 仍是占位地址，未接真实机场订阅
- `mihomo` 当前会因占位订阅返回 `404 Not Found`
- 订阅、出口组、监听器仍缺少结构化 CRUD 表单
- 当前后台虽然已多页面化，但系统配置页仍以 JSON 编辑器为主
- 预检仍未覆盖：
  - 监听端口占用探测
  - 真实订阅联通性批量探测
  - systemd active / enabled 状态读取
- 真实 SOCKS5 出口链路尚未基于真实订阅做完整出网验证
- 日志、指标和长期运维能力仍不足


## 下一步建议

1. 把 `/etc/proxyrelay/proxyrelay.yaml` 中的订阅 URL 替换为真实地址
2. 用真实订阅验证 provider 刷新和节点命中
3. 用真实节点验证至少一个 SOCKS5 端口可稳定出网
4. 把“组切换”从当前运行态可调用推进为真实业务链路可验证
5. 为订阅、出口组、监听器补结构化表单，而不是继续主要依赖 JSON 编辑器
6. 扩展预检，补 systemd 状态、端口占用和真实订阅探测
7. 增加运行日志、Prometheus 指标和更完整的部署脚本


## 当前主线方向

当前主线已经明确，不再继续在“方案选择”上打转，而是按下面顺序推进：

1. 保持 `V0.5` Ubuntu 联调能力稳定
2. 用真实订阅跑通真实出口链路
3. 补结构化配置管理与多页面后台细化
4. 增强日志、预检、指标和部署收口
5. 再进入 `V1.0` 的稳定性和生产化建设
