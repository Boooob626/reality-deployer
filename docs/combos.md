# 协议组合

向导支持以下三种组合，可**多选并存**（各自独立入站/端口）。

## 1. VLESS + Vision + REALITY / TCP（主力）

- **Vision flow**：`flow: "xtls-rprx-vision"`，消除外层 TLS 双重加密，官方称"数倍甚至更高性能"。
- **REALITY**：服务端 `reality.settings`，`target`（v26.1.18 起 `dest` 已更名）+ `serverNames` + `privateKey` + `shortIds`。
- **传输层**：raw TCP（Vision 与 XHTTP 互斥，故不走 XHTTP）。
- **REALITY 源**三选一：
  - `own_domain`：`target` 指向 Angie 的 `127.0.0.1:8443`，借用自有域名的真证书 → 最强伪装。
  - `curated`：从 `data/reality-curated.txt` 选已验证可用的外部站点。
  - `custom`：用户输入 SNI，实时校验 `data/reality-banlist.txt`（Apple 等命中即拒）。

## 2. VLESS + Vision + TLS（自有 ACME 证书）

- 不走 REALITY；Xray 直接用 Angie 签发的 ACME 证书做 TLS 终结。
- 仍用 Vision flow + raw TCP。
- 适用：REALITY 被针对时的备选；优点是真证书、缺点是需自有域名 + 端口 443 与 REALITY 组合互斥（向导会提示择一占用 443）。

## 3. Hysteria2

- QUIC/UDP，独立端口；入站结构依据 Xray PR #5679：`protocol:"hysteria"` + `version:2`、`network:"hysteria"`、salamander 在 `streamSettings.finalmask.udp`、`alpn:["h3"]`。
- 可选 Salamander obfs（`finalmask.udp[].type=salamander`）。
- **端口跳跃**：Xray 入站监听单端口；`scripts/lib/hy2hop.sh` 用 iptables/ip6tables 把 WAN 上 UDP 区间 REDIRECT 到监听端口（独立 chain `RD_HY2`，幂等应用/卸载），并持久化到 `/etc/iptables/rules.v{4,6}` + `netfilter-persistent` 开机恢复。ufw 同时放行该区间。
- 自签证书（`openssl` 生成），客户端 `insecure=1`。
- 丢包严重/晚高峰时的备选落地。

## 路由预设（所有组合共享）

- `block_cn`（默认）：`geosite:cn` + `geoip:cn` → `block`。
- `bypass_cn`：上述 → `direct`（不走代理）。
- `none`：不附加区域规则。
- 私网 `geosite:private` → `direct`（始终）；可选 `geosite:category-ads-all` → `block`。

## 443 端口仲裁

VLESS-REALITY 与 VLESS-TLS 都要 443，不能并存于同一 443。向导逻辑：
- 若两者都选 → 提示，默认 REALITY 占 443，VLESS-TLS 让位或改用 REALITY 内部回源。
- 推荐组合：**REALITY(443/tcp) + Hysteria2(udp)**。
