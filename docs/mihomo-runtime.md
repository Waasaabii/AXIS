# Mihomo Runtime 说明

## 角色划分

在当前实现中：

- `proxyrelayd` 负责高层配置、登录、状态聚合、订阅预览、运行预检和 `mihomo.yaml` 渲染
- `mihomo` 负责真实代理协议承载、SOCKS5 流量转发和 `external-controller`


## 运行目录建议

推荐在服务器上使用如下目录：

```text
/etc/proxyrelay/
  proxyrelay.yaml

/var/lib/proxyrelay/
  runtime/
    mihomo.yaml
    mihomo.last-good.yaml
    control-state.json
    providers/
```


## 当前接入方式

当前控制面会做这些事：

1. 读取 `proxyrelay.yaml`
2. 渲染出 runtime 下的 `mihomo.yaml`
3. 记录 `mihomo.last-good.yaml`
4. 记录 `control-state.json`
5. 在 managed 模式下尝试把最新配置下发给运行中的 controller


## render-only 与 managed

### render-only

适合：

- 本地只做控制面开发
- 暂时不接入真实 `mihomo`

特点：

- 可以登录后台
- 可以渲染配置
- 可以做订阅测试与节点预览
- 不要求 controller 可达


### managed

适合：

- Ubuntu 联调
- 真实运行环境

特点：

- 需要检测到 `mihomo` 二进制
- 需要 controller 可达
- 支持运行态 reload
- 支持记录运行态下发结果


## 当前已验证的 runtime 能力

在当前 Ubuntu 主机上已经验证过：

- `mihomo` 可启动
- `controller` 可达
- `POST /api/reload` 可触发下发
- `mihomo /version` 可读
- `mihomo /configs` 可读
- 运行预检可全绿

这意味着 runtime 链路本身已经通了。


## 当前限制

当前剩余限制主要不在 runtime 自身，而在业务配置层：

- 示例订阅地址仍是占位值
- provider 拉取会返回 `404 Not Found`
- 真实订阅和真实出网能力尚未完全验证

换句话说：

- **runtime 已接通**
- **真实业务数据还没接通**


## 推荐端口边界

公网开放：

- SOCKS5 监听端口，例如 `10801-10803`

建议仅本机访问：

- `proxyrelayd` API，例如 `127.0.0.1:11234`
- `mihomo external-controller`，例如 `127.0.0.1:11235`


## 本地联调方式

当前仓库已经提供本地启动脚本：

```bash
cd /home/ubuntu/ProxyRelay
./deploy/start-local.sh
```

它会按固定顺序：

1. 停掉已有 systemd 服务
2. 生成本地 managed 配置
3. 先启动 `mihomo`
4. 再启动控制面

适合当前在本地快速验证 `mihomo + ProxyRelay` 双进程联调。
