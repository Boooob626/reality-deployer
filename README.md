# Reality Deployer

交互式 Xray-core + Angie 自动部署脚本（Go 向导 + Bash 执行层）。在 Debian/Ubuntu VPS 上一键完成：交互式收集所有选项 → 安装 Xray + Angie → ACME 签发证书 → 配置 ufw → 写 systemd → 导出客户端连接信息。

## 设计哲学：Go 是大脑，Bash 是双手

| 层 | 语言 | 职责 |
|---|---|---|
| 向导 / 校验 / 配置渲染 | **Go** | 交互提示、DNS 与端口校验、Xray JSON / Angie conf / systemd 模板渲染、生成 `staging/` + `apply.env` + `manifest.json` |
| 系统变更 | **Bash** | apt 安装、Angie 官方源、ufw、systemctl、文件落位、ACME 触发 |

数据契约：Go 把决策写成 `manifest.json`（声明式记录，供卸载/重配）、`apply.env`（安装参数）与 `staging/`（待落位配置）；`scripts/install.sh` 负责应用这些产物。详见 [docs/architecture.md](docs/architecture.md)。

## 支持的协议组合（向导中可多选并存）

| 组合 | 说明 |
|---|---|
| **VLESS + Vision + REALITY / TCP** | 主力。REALITY 借用真实站点 TLS 握手，Vision 消除双重加密 |
| **VLESS + Vision + TLS** | 不走 REALITY，直接用 Angie 签发的 ACME 证书 + Vision |
| **VLESS + XHTTP + TLS** | 新客户端可选项。XHTTP 自带 XMUX/HTTP2 复用、header padding 与多模式上行，适合追求低延迟与更强 HTTP 伪装的线路 |
| **Hysteria2** | QUIC/UDP，Brutal 拥塞控制，可选端口跳跃，丢包严重时的备选落地 |

详见 [docs/combos.md](docs/combos.md)。

## 关键约束（已内置）

- **REALITY 源三选一**：自有域名经 Angie（ACME 真证书，最强伪装）/ 精选外部目标 / 自定义 SNI
- **封禁 Apple** 作为 REALITY 目标 SNI（Apple 会发法律通知 + 易被标记），见 `data/reality-banlist.txt`
- **路由默认屏蔽 CN/RU 目标**（`geosite:cn` + `domain:ru` + `geoip:cn/ru` → block），可切 bypass-direct
- **始终拦截私网 IP 与明文 BitTorrent**，避免 VPS 被当内网跳板或滥用出口；Xray sniffing 使用 `routeOnly` + `AsIs`，保留域名/IP 分流但不额外发起 DNS 解析
- **可选低延迟内核调优**：写入 fq、BBR（若内核支持）与 TCP Fast Open；卸载时会回收该 sysctl 文件
- ACME 用 **HTTP-01**（开放 80 端口）；Web 服务器用 **Angie**（内置 ACME，不用 Certbot/Caddy）

## 快速开始

> **仅运行在 Linux VPS（Debian/Ubuntu）上。** 一键脚本从 GitHub Release 下载**预编译二进制**（含 `scripts/`），解压即用——**不在 VPS 上编译，无需 make / Go 工具链 / scp**。

### 方式一：一键（curl，推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/Boooob626/reality-deployer/main/install.sh | sudo bash
```

### 方式二：手动下载 Release

从 [Releases](https://github.com/Boooob626/reality-deployer/releases) 下载对应架构的 `reality-deployer-linux-<arch>.tar.gz`，解压后运行：

```bash
mkdir -p /opt/reality-deployer
tar -xzf reality-deployer-linux-amd64.tar.gz -C /opt/reality-deployer
cd /opt/reality-deployer && sudo ./deploy.sh
```

向导会依次询问：域名（校验 DNS→本机 IP）、ACME 邮箱、协议组合、REALITY 源、XHTTP/Hysteria2 选项、路由预设、低延迟调优、SSH 端口，确认后自动部署并打印 `vless://` / `hysteria2://` 连接链接。

> 开发者：从源码构建 Release tarball 用 `make package`（仅用于打包发布，VPS 上不需要）。

## 子命令

```bash
reality-deployer wizard      # 交互式向导（默认）
reality-deployer apply       # 执行 install.sh（安装/配置/启动/签发证书）
reality-deployer uninstall   # 执行 uninstall.sh（停服务/删配置/回收 ufw+iptables）
reality-deployer status      # 查看当前部署（读 manifest.json）
reality-deployer export      # 重新导出客户端连接链接（+ 二维码）
```

### Reconfigure 与二维码

- **Reconfigure**：在已有部署的 VPS 上再次运行 `wizard`，会自动检测既有 `manifest.json` 并把各提示的**默认值预填为当前配置**（回车=保留，输入新值=修改），且会沿用旧的 UUID/REALITY 密钥，客户端无需重配。
- **二维码**：`export` 在安装了 `qrencode` 时会为每条链接打印终端二维码（`apt install qrencode`）。

## 验证

```bash
xray run -test -c /usr/local/etc/xray/config.json   # 配置语法
angie -t                                              # Angie 配置语法
ufw status verbose                                    # 仅 22/80/443/hy2-udp
curl -vk https://<domain>/                            # REALITY own_domain 时返回 decoy 站点 + 有效证书
```

## 目录结构

```
reality-deployer/
├── deploy.sh                # root 入口
├── cmd/deploy/main.go       # CLI 入口（wizard/apply/uninstall/status/export）
├── internal/
│   ├── wizard/              # 交互提示 + 校验 + 流程编排
│   ├── combo/               # VLESS-REALITY / VLESS-TLS / VLESS-XHTTP / Hysteria2 inbound 生成
│   ├── reality/             # 目标源解析 + Apple 封禁 + X25519 密钥
│   ├── routing/             # CN/RU block / bypass / 广告 / 滥用防护 预设
│   ├── config/              # 渲染 xray.json / angie conf / systemd / ufw / apply.env
│   ├── manifest/            # 唯一事实来源（Save/Load）
│   ├── link/                # vless:// / hysteria2:// 链接
│   ├── paths/               # 系统路径常量
│   └── assets/files/        # 模板 + decoy + banlist/curated（go:embed）
├── scripts/                 # install.sh / uninstall.sh / lib/（系统变更层）
└── docs/                    # architecture.md / combos.md
```

## 范围之外（YAGNI）

TUI 全屏界面、多用户订阅/计费、DNS-01/泛域名、RHEL 系/Arch 支持、Xray 自动更新。

## License

MIT，见 [LICENSE](LICENSE)。
