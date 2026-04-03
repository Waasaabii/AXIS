# AGENTS

## 目标

本文件定义 AXIS 仓库内所有代理执行时必须遵守的路径模型与产出规则。

这不是建议，是硬约束。

## 路径总原则

AXIS 统一采用 Electron 风格的 `userData` 目录策略。

只允许两类默认根目录：

- 正式运行：`userData/AXIS`
- 开发 profile：`userData/AXIS/dev`

默认映射：

- macOS：`~/Library/Application Support/AXIS`
- Windows：`%APPDATA%/AXIS`
- Linux：`~/.config/AXIS`

开发 profile 在正式根目录下追加 `/dev`。

## 覆盖规则

只允许两种显式覆盖：

- `AXIS_HOME`
  - 覆盖整个数据根目录
- `PROXYRELAY_CONFIG`
  - 覆盖配置文件路径

除了这两个显式覆盖外，禁止任何形式的隐式回退和路径猜测。

禁止：

- 根据当前工作目录猜配置路径
- 找不到文件时自动改写到仓库根目录
- 找不到文件时自动改写到应用同级目录
- 在不同入口之间使用不同的默认数据根目录

## 运行态目录结构

运行态根目录内部结构固定为：

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

代理修改代码时，不得擅自引入新的散落目录，除非先同步更新 README、AGENTS、CLAUDE。

## 构建产物与运行态产物

必须明确区分：

- 构建产物
  - 允许留在仓库内
  - 例如 `./desktop/build`、`./.tmp/desktop`
- 运行态产物
  - 必须进入 `AXIS_HOME`
  - 包括配置、数据库、缓存、渲染文件、updater 状态、日志

禁止再把运行态产物默认写到：

- `./runtime`
- `./runtime/dev`
- 仓库根目录下任意临时文件
- 用户目录中的其他随机位置

## 各入口必须遵守同一模型

以下入口必须使用同一套路径规则：

- `cmd/axis`
- `desktop/main.go`
- `pnpm dev`
- `pnpm dev:desktop`
- `pnpm reset:setup`

如果修改其中任一入口的路径逻辑，必须同步检查其余入口是否仍然一致。

## 文档同步要求

涉及以下内容变更时，必须同步更新文档：

- 默认数据目录
- 开发 profile 目录
- 覆盖变量
- `reset:setup` 行为
- 新增或删除运行态子目录
- 会话失效或 reset 行为

必须同步的文件：

- `README.md`
- `AGENTS.md`
- `CLAUDE.md`

## 提交前最低检查

凡是修改路径模型或运行态目录逻辑，至少验证：

- `go test ./...`
- `pnpm lint`
- `pnpm --filter @axis/frontend build`

如果涉及桌面入口，再补：

- `pnpm run build:desktop`
