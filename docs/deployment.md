# ProxyRelay 单机部署说明

## 1. 目标形态

部署完成后，服务器上会有两个主要进程：

- `proxyrelayd`：Node.js 控制面
- `mihomo`：数据面代理内核

客户端只连接 SOCKS5 端口，不直接接触机场订阅。


## 2. 推荐目录

建议使用如下目录布局：

```text
/opt/proxyrelay
/etc/proxyrelay/proxyrelay.yaml
/var/lib/proxyrelay/runtime/
/var/lib/proxyrelay/logs/
```


## 3. 初始化目录与权限

建议使用单独系统用户：

```bash
sudo useradd --system --home /opt/proxyrelay --shell /usr/sbin/nologin proxyrelay
sudo mkdir -p /opt/proxyrelay /etc/proxyrelay /var/lib/proxyrelay/runtime /var/lib/proxyrelay/logs
sudo chown -R proxyrelay:proxyrelay /opt/proxyrelay /etc/proxyrelay /var/lib/proxyrelay
sudo chmod 700 /etc/proxyrelay /var/lib/proxyrelay
```


## 4. 部署应用文件

将当前项目部署到：

```text
/opt/proxyrelay
```

至少需要这些内容：

- `src/`
- `public/`
- `package.json`
- `node_modules/`
- `examples/proxyrelay.yaml`
- `deploy/*.service`
- `deploy/install-ubuntu.sh`


## 5. 使用安装脚本初始化

如果代码已经放到目标机器上，可以直接执行：

```bash
sudo APP_DIR=/opt/proxyrelay /opt/proxyrelay/deploy/install-ubuntu.sh
```

这个脚本会完成：

- 创建 `proxyrelay` 用户与基础目录
- 生成 `/etc/proxyrelay/proxyrelay.yaml`（若不存在）
- 安装 `proxyrelayd.service` 与 `mihomo.service`
- 执行 `systemctl daemon-reload`

脚本**不会**替你写入真实订阅、密码或 `secret`，这些仍需要手动修改。


## 6. 部署 `mihomo`

需要提前把 `mihomo` 二进制放到：

```text
/usr/local/bin/mihomo
```

并确保：

```bash
sudo chmod +x /usr/local/bin/mihomo
```


## 7. systemd 启动方式

安装完配置和 service 文件之后，可以直接启用：

```bash
sudo systemctl enable --now proxyrelayd
sudo systemctl enable --now mihomo
```

查看状态：

```bash
sudo systemctl status proxyrelayd
sudo systemctl status mihomo
```

查看日志：

```bash
journalctl -u proxyrelayd -f
journalctl -u mihomo -f
```


## 8. 手工启动方式

如果你不想先走 systemd，也可以手工启动。

### 8.1 先启动 `mihomo`

```bash
/usr/local/bin/mihomo -d /var/lib/proxyrelay/runtime -f /var/lib/proxyrelay/runtime/mihomo.yaml
```

### 8.2 再启动控制面

```bash
cd /opt/proxyrelay
PROXYRELAY_CONFIG=/etc/proxyrelay/proxyrelay.yaml node src/index.js serve
```


## 9. 多页面后台入口

当前后台已经是多页面结构，常用入口如下：

- `/dashboard.html`
- `/status.html`
- `/subscriptions.html`
- `/interfaces.html`
- `/system.html`
- `/events.html`

如果控制面监听在 `127.0.0.1:11234`，可通过：

```text
http://127.0.0.1:11234/dashboard.html
```

如果你明确改成 `0.0.0.0`，则可通过服务器地址访问。


## 10. 部署后验证

### 10.1 验证控制面

```bash
curl http://127.0.0.1:11234/api/session
```

### 10.2 验证运行预检

```bash
sudo -u proxyrelay env PROXYRELAY_CONFIG=/etc/proxyrelay/proxyrelay.yaml \
  node /opt/proxyrelay/src/index.js preflight
```

如果只想拿 JSON：

```bash
sudo -u proxyrelay env PROXYRELAY_CONFIG=/etc/proxyrelay/proxyrelay.yaml \
  node /opt/proxyrelay/src/index.js preflight --json
```

### 10.3 验证端口

```bash
ss -lntp | rg ':(11234|11235|10801|10802|10803)\b'
```

### 10.4 验证 controller

```bash
curl -H "Authorization: Bearer <your-secret>" http://127.0.0.1:11235/version
curl -H "Authorization: Bearer <your-secret>" http://127.0.0.1:11235/configs
```


## 11. 防火墙建议

公网开放：

- `10801`
- `10802`
- `10803`

建议仅本机访问：

- `11234`：ProxyRelay 控制面
- `11235`：Mihomo controller

如果为了临时测试把控制面改成公网监听，测试完成后建议恢复为本机监听。


## 12. 上线前检查清单

- `proxyrelay.yaml` 中订阅 URL 已替换为真实地址
- `admin.password` 已替换，或改成 `password_hash`
- `runtime.external_secret` 已配置
- `mihomo` 二进制路径正确
- `preflight` 不再报告阻塞级错误
- 已验证 controller 可达
- 已验证至少一个 SOCKS5 端口可连通
- 如要公网暴露控制面，已完成额外安全防护
