# 启动、部署和维护

## 本地运行

Windows 发布包双击 `start.cmd`，或执行 `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/run.ps1`。默认访问 `http://127.0.0.1:8080/`。Linux amd64 发布包使用 `sh scripts/run.sh`。从源码构建见 README。

操作系统环境变量（直接启动不自动读取 `.env`）：

| 变量 | 默认/作用 |
| --- | --- |
| LISTEN_ADDR | `127.0.0.1:8080`，内网测试可改为 `0.0.0.0:8080` |
| DATA_DIR | 运行脚本默认项目根目录 `data/` |
| STATIC_DIR | 运行脚本默认项目根目录 `frontend/dist/` |
| ADMIN_KEY | 后台和指标的管理密钥；脚本未设置时自动生成到数据目录 |
| COOKIE_SECURE | HTTPS环境设 `true`，本地HTTP使用 `false` |
| TRUSTED_PROXIES | 可信反向代理的 IP/CIDR，逗号分隔；默认空，不信任转发头 |

例如电脑内网测试，先在 PowerShell 设置 `$env:LISTEN_ADDR='0.0.0.0:8080'` 再运行脚本。手机与电脑连接同一网络，使用电脑真实内网IP及8080端口，按系统防火墙策略放行。此环境不提供公网HTTPS或完整安装能力。

## HTTPS / Docker 模板

复制 `.env.example` 为 `.env`，设置自己的 DOMAIN 和强随机 ADMIN_KEY，然后在有 Docker Compose 的机器执行：

```sh
docker compose up --build -d
docker compose logs --tail 100
```

模板使用 Go 容器、持久卷与 Caddy TLS 反向代理。域名需解析到服务器，80/443端口可用。Docker Compose 为 Caddy 分配 `172.30.0.2`，并只将该地址配置为可信代理；应用只解析可信代理发来的 `X-Forwarded-For`。若 `172.30.0.0/24` 与主机网络冲突，应同步修改代理静态地址和 `TRUSTED_PROXIES`。游戏容器的8080端口只通过 Compose 内部网络暴露，不应另行映射到公网。Caddy 对公网 `/metrics` 返回404，监控应从受控内部网络读取服务8080的 `/metrics` 并发送 `X-Admin-Key`。模板尚未在本机实际构建、签发证书或部署验收；Linux程序仅完成交叉编译。

不要在同一个数据目录启动两个服务或两个容器实例。当前没有数据库锁或跨文件事务，不能水平扩容。登录、公开战绩 API 和操作限流均按解析后的客户端 IP 计数；直接启动默认忽略转发头，只有明确配置 `TRUSTED_PROXIES` 的来源才会采用转发链。不要把不受信任的代理或任意公网网段加入此配置。限流状态保存在单进程内存中，不适用于多实例共享或抵御大规模网络层 DDoS；可信代理部署还应保持8080端口不可被公网直连。

## 备份与升级

停止服务后备份整个 DATA_DIR，妥善保管 `admin-key.txt`，将其访问权限限制为运行账号和管理员。升级前保留旧程序与完整数据副本；替换程序和静态资源后启动，检查 `/readyz`、账号、房间和历史记录，再让用户访问。避免在服务写入中直接复制部分JSON文件作为一致备份。

用户名与密码哈希、会话摘要保存在JSON。当前没有密码重置/找回、短信找回或微信OAuth；玩家应妥善保管登录凭据。管理密钥可以通过环境变量替换，恢复运行后使用新密钥。

## 验收与监控

`/healthz` 和 `/readyz` 返回200后再验证一个真实创建/操作请求；readiness并不探测每次写盘。指标包含进程CPU累计秒数、Go堆/系统内存、协程数、在线账号、WebSocket连接、房间、请求量、命令量、存储错误及请求/命令耗时直方图。CPU为进程累计用时；Go系统内存指标不等于操作系统RSS。

浏览器后台状态需管理密钥。对外部署建议独立管理网络、TLS、反向代理访问控制与日志轮转。当前版本尚未完成公网并发对局、备份恢复演练、Linux运行、Docker和真实iOS/Android验收，请按测试记录逐项补做。
