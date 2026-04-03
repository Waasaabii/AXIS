# CLAUDE

## AXIS 路径模型全貌

AXIS 当前统一采用 Electron 风格的 `userData` 目录策略。

核心规则只有一句话：

> 所有默认运行态产出都必须落在同一个稳定数据根目录下。

## 默认数据根目录

正式运行：

- macOS：`~/Library/Application Support/AXIS`
- Windows：`%APPDATA%/AXIS`
- Linux：`~/.config/AXIS`

开发 profile：

- macOS：`~/Library/Application Support/AXIS/dev`
- Windows：`%APPDATA%/AXIS/dev`
- Linux：`~/.config/AXIS/dev`

## 目录结构

正式运行和开发 profile 使用同一套内部结构：

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

## 显式覆盖

只允许两种显式覆盖：

- `AXIS_HOME`
- `PROXYRELAY_CONFIG`

没有显式覆盖时，任何入口都不得改用别的默认目录。

## 禁止事项

禁止：

- 依赖 `cwd` 决定默认配置路径
- 在仓库根目录默认创建运行态文件
- 在桌面模式和 CLI 模式使用不同默认根目录
- 通过“找不到就回退”的方式写到其他目录

## 构建产物与运行态产物

构建产物允许保留在仓库：

- `./desktop/build`
- `./.tmp/desktop`

运行态产物必须进入 `AXIS_HOME`：

- 配置
- runtime 数据
- updater 状态
- 日志
- 开发 profile 痕迹

## 入口一致性

以下入口必须共享同一套路径模型：

- `cmd/axis`
- `desktop/main.go`
- `scripts/dev.mjs`
- `scripts/desktop-dev.mjs`
- `scripts/reset-setup.mjs`

任何人修改其中一处，都必须检查其他入口是否仍一致。

## Reset 行为

`reset:setup` 的语义不是只清理 runtime。

它必须同时完成：

- 清空初始化资源
- 清空运行态痕迹
- 让当前 profile 的既有会话失效

否则用户会看到“已经 reset，但重新打开还是直接登录”的错误体验。

## 文档一致性

路径模型发生变化时，必须同步更新：

- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
