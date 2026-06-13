# 协议组合

向导支持以下组合，可**多选并存**（各自独立入站/端口；443/tcp 组合会仲裁）。

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

## 3. VLESS + XHTTP + TLS（新客户端低延迟选项）

- 不走 Vision flow；使用 `network:"xhttp"` + Angie ACME 证书。
- XHTTP 服务端默认监听 TCP 443 的 H1/H2，`mode:"auto"`；需要兼容中间盒时可在向导里切到 `packet-up`，需要流式上行时可切 `stream-up`。
- `xhttpSettings.extra` 内置 `xPaddingBytes:"100-1000"`、`scMaxBufferedPosts:30`、`scStreamUpServerSecs:"20-80"`，用于减少固定 header 长度特征并保持 stream-up 长连接活性。
- 导出的 `vless://` 链接会带一个轻量 XMUX `extra`：`maxConcurrency:"16-32"`、`hMaxRequestTimes:"600-900"`、`hMaxReusableSecs:"1800-3000"`。这是客户端侧复用控制，不是经典 `mux.cool`。
- 使用 XHTTP 时不要再启用客户端的 `mux.cool`；官方 XHTTP 文档也建议让 XMUX 接管 H2/H3 复用。
- 适用：客户端足够新、想要 HTTP 传输层伪装/复用能力；代价是兼容面比 REALITY/Vision 更依赖客户端实现。

## 4. Hysteria2

- QUIC/UDP，独立端口；入站结构依据 Xray PR #5679：`protocol:"hysteria"` + `version:2`、`network:"hysteria"`、salamander 在 `streamSettings.finalmask.udp`、`alpn:["h3"]`。
- 可选 Salamander obfs（`finalmask.udp[].type=salamander`）。
- **端口跳跃**：Xray 入站监听单端口；`scripts/lib/hy2hop.sh` 用 iptables/ip6tables 把 WAN 上 UDP 区间 REDIRECT 到监听端口（独立 chain `RD_HY2`，幂等应用/卸载），并持久化到 `/etc/iptables/rules.v{4,6}` + `netfilter-persistent` 开机恢复。ufw 同时放行该区间。
- 自签证书（`openssl` 生成），客户端 `insecure=1`。
- 丢包严重/晚高峰时的备选落地。

## 路由预设（所有组合共享）

- `block_cn_ru`（默认）：`geosite:cn` + `domain:ru` + `geoip:cn` + `geoip:ru` → `block`。
- `block_cn`：`geosite:cn` + `geoip:cn` → `block`。
- `bypass_cn_ru`：CN/RU 目标 → `direct`（走 VPS 自身出口）。
- `bypass_cn`：上述 → `direct`（不走代理）。
- `none`：不附加区域规则。
- 私网 `geoip:private` → `block`（始终），避免 VPS 被当内网跳板或云元数据探针。
- 明文 BitTorrent `protocol:["bittorrent"]` → `block`（始终），降低出口投诉和滥用风险。
- 入站 sniffing 使用 `routeOnly:true`，路由 `domainStrategy:"AsIs"`，可按域名/IP 分流但不额外发起 DNS 解析。
- 可选 `geosite:category-ads-all` → `block`。

## 443 端口仲裁

VLESS-REALITY、VLESS-TLS 与 VLESS-XHTTP 都要 443/tcp，不能并存于同一 443。向导逻辑：
- 若多个 443 组合都选 → 提示择一，默认 REALITY 占 443。
- 推荐保守组合：**REALITY(443/tcp) + Hysteria2(udp)**。
- 想尝试新传输层时，用 **XHTTP(443/tcp) + Hysteria2(udp)**，并确认客户端支持 XHTTP/XMUX。
