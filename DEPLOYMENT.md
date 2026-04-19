# Sub2Balance 服务器部署指南

本文档介绍两种部署方式，推荐使用 **Docker Compose**。后半部分是反向代理、备份、升级、排障等运维内容。

---

## 一、环境要求

| 项目 | 最低 | 推荐 |
| --- | --- | --- |
| OS | Linux x86_64 / arm64 | Ubuntu 22.04+ / Debian 12+ |
| CPU | 1 核 | 2 核 |
| 内存 | 512 MB | 1 GB |
| 磁盘 | 1 GB | 10 GB（视审计日志量） |
| 网络 | 能访问 `ghcr.io` 和 Sub2API 上游 | 同左 |

**软件依赖（Docker 方案）**

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# 重新登录使 docker 组生效
```

Docker 20.10+ 自带 compose v2，命令为 `docker compose`（中间有空格）。

---

## 二、方案一：Docker Compose 部署（推荐）

### 1. 创建部署目录

```bash
mkdir -p /opt/sub2balance && cd /opt/sub2balance
```

### 2. 下载配置模板

```bash
# 三份模板（可以直接从仓库 raw 链接 curl，也可以 git clone 后 cp）
BASE=https://raw.githubusercontent.com/asdwsxzc123/sub2balance/master
curl -O $BASE/docker-compose.yml
curl -o config.yaml      $BASE/config.yaml.example
curl -o .env             $BASE/.env.example
```

### 3. 编辑 `.env`

```bash
nano .env
```

必须改的三项：

```dotenv
# 用 openssl rand -hex 32 生成
JWT_SECRET=<换成 64 位随机串>

# 第一次启动时会在 users 表里创建这个 admin 账号
ADMIN_EMAIL=admin@yourcompany.com
ADMIN_PASSWORD=<强密码>
```

可选：如果你的宿主机用户 UID 不是 1000，加上：

```dotenv
SUB2BALANCE_UID=$(id -u)
SUB2BALANCE_GID=$(id -g)
```

### 4. 准备 `data` 目录

容器内以 UID 1000 运行，宿主机的 `./data` 目录必须可写：

```bash
mkdir -p data

# 方式 A：把 data 交给 1000:1000（默认容器 UID）
sudo chown 1000:1000 data

# 方式 B：你在 .env 设置了自定义 UID/GID，那就 chown 到你自己的
sudo chown $(id -u):$(id -g) data
```

### 5. 检查 `config.yaml`

默认 `config.yaml` 已经能直接用，只需要确认 `server.port` 是 `8080`（和 compose 里对外映射一致）。如果你要换端口，改 compose 的 `ports:` 一侧即可，不用动 config.yaml。

### 6. 启动

```bash
docker compose pull          # 拉取最新镜像
docker compose up -d         # 后台启动
docker compose logs -f       # 观察启动日志（Ctrl+C 退出不会停容器）
```

看到 `Server starting on :8080` 就成功了。

### 7. 首次登录 + 配置 Sub2API

1. 浏览器打开 `http://<服务器IP>:8080`
2. 用 `.env` 里的 `ADMIN_EMAIL` / `ADMIN_PASSWORD` 登录
3. 进入 **系统设置**（`/admin/settings`），填入：
   - **Base URL**：你的 Sub2API 实例地址
   - **API Key**：有 admin 权限的 key
4. 点 **测试连接**，成功后保存
5. 到 **分组定价**（`/admin/group-prices`）配置各订阅组的单价
6. 顶部菜单右侧修改 admin 密码（强烈建议）

### 8. 锁定默认端口

生产环境不要直接把 8080 暴露到公网。继续往下看 **反向代理 + HTTPS** 章节。

---

## 三、方案二：二进制部署（无 Docker）

适合无法装 Docker 的环境。要求系统能运行 glibc/musl 的 Linux。

### 1. 下载对应平台的 release

```bash
# 在 https://github.com/asdwsxzc123/sub2balance/releases 找到最新 tag
VERSION=v1.0.0
ARCH=linux-amd64   # 或 linux-arm64

curl -LO https://github.com/asdwsxzc123/sub2balance/releases/download/$VERSION/sub2balance-$ARCH.tar.gz
tar xzf sub2balance-$ARCH.tar.gz
```

解压出来会得到：`sub2balance-linux-amd64`（二进制）、`config.yaml.example`、`.env.example`、`README.md`。

### 2. 初始化配置

```bash
cp config.yaml.example config.yaml
cp .env.example .env
nano .env    # 改 JWT_SECRET / ADMIN_EMAIL / ADMIN_PASSWORD
```

### 3. 跑起来（前台测试）

```bash
set -a; source .env; set +a
./sub2balance-linux-amd64
```

访问 `http://<IP>:8080` 确认能登录。Ctrl+C 停掉。

### 4. 注册为 systemd 服务

```bash
sudo tee /etc/systemd/system/sub2balance.service > /dev/null <<EOF
[Unit]
Description=Sub2Balance
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$(pwd)
EnvironmentFile=$(pwd)/.env
ExecStart=$(pwd)/sub2balance-linux-amd64
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now sub2balance
sudo systemctl status sub2balance
```

查日志：`sudo journalctl -u sub2balance -f`

---

## 四、反向代理 + HTTPS

### Caddy（最简单，自动签证书）

```caddyfile
# /etc/caddy/Caddyfile
sub2balance.yourdomain.com {
    reverse_proxy 127.0.0.1:8080
}
```

```bash
sudo systemctl reload caddy
```

Caddy 会自动从 Let's Encrypt 申请并续期证书。

### Nginx + certbot

```nginx
# /etc/nginx/sites-available/sub2balance
server {
    listen 80;
    server_name sub2balance.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/sub2balance /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d sub2balance.yourdomain.com
```

### 防火墙

```bash
# Ubuntu/Debian
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
# 不要开放 8080！让它只监听本机
```

如果要求应用只监听 127.0.0.1，编辑 `docker-compose.yml`：

```yaml
    ports:
      - "127.0.0.1:8080:8080"   # 只绑定到本机
```

---

## 五、运维

### 升级版本

**Docker 方案：**

```bash
cd /opt/sub2balance
docker compose pull
docker compose up -d
```

要锁定版本，编辑 `.env`：

```dotenv
SUB2BALANCE_IMAGE=ghcr.io/asdwsxzc123/sub2balance:v1.2.3
```

**二进制方案：** 重新下载新版 tar.gz，替换同路径下的二进制，然后 `sudo systemctl restart sub2balance`。

### 备份

数据全部在 `data/sub2balance.db`（一个 SQLite 文件）。

```bash
# 定时备份（crontab）
0 3 * * * cd /opt/sub2balance && sqlite3 data/sub2balance.db ".backup '/backup/sub2balance-$(date +\%F).db'"

# 或更简单：直接 tar
0 3 * * * tar czf /backup/sub2balance-$(date +\%F).tar.gz -C /opt/sub2balance data/
```

**恢复**：停服务 → 把备份覆盖回 `data/sub2balance.db` → 启动。

### 日志

- Docker：`docker compose logs -f --tail=200`
- systemd：`journalctl -u sub2balance -f`
- 应用自己不写日志文件，一律走 stdout/stderr

### 健康检查

```bash
curl -fsS http://127.0.0.1:8080/health
# 返回 {"status":"ok"} 表示活着
```

Docker 镜像内置了 HEALTHCHECK，`docker ps` 的 STATUS 列会显示 `(healthy)`。

### 监控建议

- Uptime 探针打 `/health`
- 磁盘告警：`data/` 目录超过设定阈值（审计日志会累积）
- 定期用 `sqlite3 data/sub2balance.db "VACUUM;"` 回收空间（停机操作）

---

## 六、常见问题

**Q: 启动报 `Failed to read config file`**
A: `config.yaml` 没准备。Docker 方案一定要先 `curl -o config.yaml ...example` 或 `cp`。

**Q: 启动报 `JWT secret is required`**
A: `.env` 里 `JWT_SECRET` 还是占位符 `your-secret-key-here`，或者 `env_file` 没加载。确认文件在 compose 同目录。

**Q: 启动报 `permission denied` on `data/sub2balance.db`**
A: `data` 目录宿主机属主不是 1000。`sudo chown 1000:1000 data`，或在 `.env` 里设 `SUB2BALANCE_UID=$(id -u)` 后 `docker compose up -d`。

**Q: 能登录但点任何页面都报 "sub2api is not configured"**
A: 去 `/admin/settings` 填 Base URL 和 API Key 并测试。

**Q: 改了 `ADMIN_PASSWORD` 重启没生效**
A: 只有 `users` 表为空时才会用它创建初始账号。想重置：`docker compose down`，`rm data/sub2balance.db`（**会清空所有数据**），再 `up -d`。生产环境请改用 admin UI 改密码。

**Q: ARM64 服务器拉 `latest` 报 "no matching manifest"**
A: 确认用的是 master 分支之后构建的镜像（见 docker.yml，已支持 linux/arm64）。老镜像只有 amd64。

**Q: 想换端口**
A: 改 `docker-compose.yml` 的 `ports: "新端口:8080"`（左侧是宿主机端口），不用改 config.yaml。

**Q: 前端改了代码怎么重新打包**
A: 只改前端不用重新发布，本地 `cd frontend && pnpm build`，然后重新 `go build .` 或 `docker build .`。因为前端是 `//go:embed` 进二进制的。

---

## 七、快速验证清单

部署完后，按这个顺序验证：

- [ ] `curl http://127.0.0.1:8080/health` 返回 `{"status":"ok"}`
- [ ] 浏览器能打开登录页
- [ ] 用 `.env` 里的邮箱密码能登录
- [ ] `/admin/settings` 配置 Sub2API 后点测试成功
- [ ] `/admin/group-prices` 添加至少一条价格映射
- [ ] 新建一个测试工单，走完 staff 提交 → admin 审批的流程
- [ ] 防火墙只放行 80/443，8080 不对公网暴露
- [ ] 配置了每日备份 cron

---

## 八、相关链接

- 镜像仓库：`ghcr.io/asdwsxzc123/sub2balance`
- Release：https://github.com/asdwsxzc123/sub2balance/releases
- 源码：https://github.com/asdwsxzc123/sub2balance
