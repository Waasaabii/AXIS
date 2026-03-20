# ProxyRelay 技术设计方案

## 1. 文档目标

本文档基于当前 [REQUIREMENTS.md](./REQUIREMENTS.md)，给出一份与仓库现状一致
的技术设计说明。文档既描述目标架构，也记录当前已经落地的实现基线。


## 2. 当前实现快照

截至 2026-03-20，项目的真实状态如下：

- 控制面：`Node.js`
- 数据面：`mihomo`
- 前端：静态多页面后台
- 本地启动：`deploy/start-local.sh`
- Ubuntu 部署：`deploy/install-ubuntu.sh` + `systemd`
- 已验证的运行形态：
  - 仓库内配置：`config/proxyrelay.yaml`
  - 系统级配置：`/etc/proxyrelay/proxyrelay.yaml`

当前已经确认可工作的能力：

- 登录
- 配置读写
- 订阅测试与节点预览
- `mihomo.yaml` 渲染
- controller 探测
- 运行预检
- managed 模式下 reload
- 多页面后台
- Ubuntu 实机双进程联调

当前尚未完全闭环的业务能力：

- 真实订阅联调
- 真实 SOCKS5 出口链路验证
- 结构化 CRUD 表单
- 更完整的运维指标与长期运行能力


## 3. 总体结论

### 3.1 推荐架构

项目继续采用方案 B：

- **数据面**：`mihomo`
- **控制面**：`proxyrelayd`（当前为 Node.js HTTP 服务）
- **配置方式**：高层 YAML -> 渲染生成底层 `mihomo.yaml`
- **运行方式**：
  - 本地开发可走 render-only 或本地 managed
  - Ubuntu 目标环境走 systemd managed


### 3.2 设计原则

当前架构坚持以下原则：

- 不自研 VMess / VLESS / Shadowsocks 协议栈
- 让 `mihomo` 负责“真实流量与协议”
- 让控制面负责“配置抽象、状态聚合、运维动作”
- 对外暴露的稳定对象是“逻辑出口组”和“SOCKS5 入口”
- 后台页面按任务域拆开，而不是一个页面塞进所有功能


## 4. 架构分层

```text
┌─────────────────────────────────────────────────────┐
│                    Web Console                      │
│ dashboard / status / subscriptions / interfaces    │
│ system / events                                     │
└───────────────────────┬─────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────────┐
│                  ProxyRelay Control Plane           │
│ - session / status / config / providers / groups   │
│ - listeners / events / rendered-config             │
│ - controller / runtime-preflight / reload          │
└───────────────────────┬─────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────────┐
│                     Runtime Assets                  │
│ - proxyrelay.yaml                                  │
│ - mihomo.yaml                                      │
│ - mihomo.last-good.yaml                            │
│ - control-state.json                               │
└───────────────────────┬─────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────────┐
│                        Mihomo                       │
│ - proxy-providers                                  │
│ - proxy-groups                                     │
│ - listeners                                        │
│ - external-controller                              │
└─────────────────────────────────────────────────────┘
```


## 5. 关键设计

### 5.1 逻辑出口组优先

端口不直接永久绑定单个原始节点，而是绑定逻辑出口组。

例如：

- `10801 -> egress-hk-manual`
- `10802 -> egress-jp-manual`
- `10803 -> egress-us-auto`

好处：

- 端口语义稳定
- 机场节点改名不会直接冲击对外接口
- 手动与自动模式都能落在同一抽象上


### 5.2 控制面维护“期望状态”

`proxyrelayd` 不直接承载代理流量，只维护：

- 订阅源
- 逻辑出口组
- 对外监听器
- 认证信息
- 运行态状态

流量转发全部交给 `mihomo`。


### 5.3 配置文件分层

高层配置：

- `config/proxyrelay.yaml`
- `/etc/proxyrelay/proxyrelay.yaml`

运行产物：

- `runtime/mihomo.yaml`
- `runtime/mihomo.last-good.yaml`
- `runtime/control-state.json`

系统级 runtime：

- `/var/lib/proxyrelay/runtime/mihomo.yaml`
- `/var/lib/proxyrelay/runtime/mihomo.last-good.yaml`
- `/var/lib/proxyrelay/runtime/control-state.json`


### 5.4 运行模式分流

当前支持两种核心模式：

#### render-only

特点：

- 只负责渲染配置
- 不要求 controller 可达
- 适合本地纯控制面开发

#### managed

特点：

- 要求 `mihomo` 二进制存在
- 要求 `external-controller` 可达
- 支持 `POST /api/reload`
- 适合 Ubuntu 联调和真实运行环境


## 6. 控制面设计

### 6.1 API 面

当前 API 已覆盖以下功能域：

- 会话：
  - `POST /api/session`
  - `GET /api/session`
  - `DELETE /api/session`
- 总览与配置：
  - `GET /api/status`
  - `GET /api/config`
  - `PUT /api/config`
  - `GET /api/rendered-config`
- 订阅与出口：
  - `GET /api/providers`
  - `GET /api/groups`
  - `GET /api/listeners`
  - `POST /api/subscription-test`
  - `POST /api/providers/:name/refresh`
  - `POST /api/groups/:name/select`
  - `POST /api/groups/:name/healthcheck`
- 运行态：
  - `GET /api/controller`
  - `POST /api/controller/probe`
  - `GET /api/runtime-preflight`
  - `POST /api/reload`
- 审计：
  - `GET /api/events`


### 6.2 状态聚合

控制面会汇总：

- app 模式
- runtime 模式
- controller 状态
- provider / group / listener 计数
- 最近渲染时间
- 最近下发状态
- 最近事件
- 当前告警摘要


### 6.3 运行态下发

当前的运行态下发链路是：

1. 读取并校验高层配置
2. 渲染最新 `mihomo.yaml`
3. 记录 runtime 状态
4. 如果为 managed 模式，则调用 controller `/configs?force=true`
5. 把下发结果写入 `lastApplyStatus / lastApplyMessage`

这条链路已经在 Ubuntu 实机上验证过。


## 7. 前端设计

### 7.1 当前形态

前端不再使用单页堆叠式仪表盘，而是采用多页面后台：

- `dashboard.html`：总览
- `status.html`：状态
- `subscriptions.html`：订阅配置
- `interfaces.html`：对外接口配置
- `system.html`：系统配置
- `events.html`：事件日志


### 7.2 页面职责

#### 总览

负责：

- 状态卡
- 系统摘要
- 重点提醒
- 快速跳转入口

#### 状态

负责：

- controller 状态
- runtime preflight
- 运行健康相关排查

#### 订阅配置

负责：

- 订阅测试
- provider 列表
- 逻辑出口组列表

#### 对外接口配置

负责：

- listener 列表
- 对外入口与出口映射说明

#### 系统配置

负责：

- 配置编辑器
- 渲染结果查看

#### 事件日志

负责：

- 最近事件
- 审计与操作回溯


### 7.3 前端风格

当前 UI 基调已切换为：

- 灰白主调
- 系统正常字体栈
- 左侧固定导航
- 多页面真实跳转

这样可以让后台更接近正常运维台，而不是概念演示页。


## 8. 本地运行设计

### 8.1 仓库默认运行

默认配置文件：

- `config/proxyrelay.yaml`

默认特点：

- 监听 `127.0.0.1:8787`
- 默认仍可保持 render-only 导向
- 适合前端、接口、本地配置链路开发


### 8.2 本地 managed 启动脚本

当前已提供：

- `deploy/start-local.sh`

作用：

- 停掉已有 systemd 服务
- 生成本地托管配置
- 先拉起 `mihomo`
- 再拉起 `proxyrelayd`
- 日志统一输出到当前终端

这是当前本地联调 `mihomo + ProxyRelay` 的推荐方式。


## 9. Ubuntu 部署设计

### 9.1 目录形态

```text
/opt/proxyrelay
/etc/proxyrelay/proxyrelay.yaml
/var/lib/proxyrelay/runtime/
```

### 9.2 服务形态

当前继续使用双 service：

- `proxyrelayd.service`
- `mihomo.service`

原因：

- 职责清晰
- 调试直接
- 便于分别看日志和生命周期


### 9.3 当前实机验证结论

已在当前 Ubuntu 环境验证：

- `mihomo` 已安装
- systemd unit 可启停
- controller 可达
- reload 可下发
- `/version` 可读
- `/configs` 可读
- preflight 可全绿

当前未闭环的不是部署，而是真实订阅链路。


## 10. 当前剩余风险

### 10.1 业务链路风险

- 示例订阅 URL 仍是占位地址
- provider 拉取会返回 `404`
- 真实出口链路尚未验证

### 10.2 控制面易用性风险

- 配置仍主要靠 JSON 编辑器
- 没有结构化 CRUD
- 页面虽然已拆分，但仍需要继续细化表格与表单体验

### 10.3 运维能力风险

- 日志能力还比较基础
- 缺 Prometheus 指标
- preflight 覆盖面仍不够完整


## 11. 下一步技术方向

1. 接入真实订阅，验证真实 provider 刷新
2. 验证真实 listener -> group -> provider -> node 出口链路
3. 补结构化 CRUD，而不是继续主要依赖 JSON 编辑器
4. 增加日志、指标和更完整的部署收口
5. 继续优化多页面后台的运维信息表达
