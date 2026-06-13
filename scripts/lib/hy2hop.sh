#!/usr/bin/env bash
# Hysteria2 端口跳跃：把 WAN 上 UDP 区间的包 REDIRECT 到监听端口。
# Xray hysteria2 入站只监听单端口；客户端在区间内跳跃，iptables 命中后重定向。
# 用独立 chain RD_HY2 便于幂等应用与卸载。依据 apply.env 的 HY2_PORT / HY2_PORTHOP。

hy2_default_iface() {
  ip -o -4 route show to default 2>/dev/null | awk '{print $5; exit}'
}

hy2hop_apply() {
  if [ "${ENABLE_HYSTERIA2:-0}" != 1 ]; then
    return 0
  fi
  if [ -z "${HY2_PORTHOP:-}" ]; then
    log "Hysteria2 未配置端口跳跃区间，跳过 iptables"
    return 0
  fi

  command -v ip >/dev/null 2>&1 || pkg_ensure iproute2
  command -v iptables >/dev/null 2>&1 || pkg_ensure iptables

  local iface port rng iarg
  iface="$(hy2_default_iface)"
  port="${HY2_PORT:-0}"
  rng="${HY2_PORTHOP}" # 形如 20000:50000，iptables --dport 接受冒号区间
  if [ "$port" = "0" ]; then
    warn "HY2_PORT 缺失，跳过端口跳跃"
    return 0
  fi
  iarg=""
  [ -n "$iface" ] && iarg="-i $iface"

  for IPT in iptables ip6tables; do
    command -v "$IPT" >/dev/null 2>&1 || continue
    "$IPT" -t nat -N RD_HY2 2>/dev/null || true
    "$IPT" -t nat -F RD_HY2 2>/dev/null || true
    "$IPT" -t nat -A RD_HY2 -p udp -j REDIRECT --to-ports "$port" 2>/dev/null || true
    # 跳转规则幂等：先删后增
    # shellcheck disable=SC2086
    "$IPT" -t nat -D PREROUTING $iarg -p udp --dport "$rng" -j RD_HY2 2>/dev/null || true
    # shellcheck disable=SC2086
    "$IPT" -t nat -A PREROUTING $iarg -p udp --dport "$rng" -j RD_HY2 2>/dev/null || true
  done

  hy2hop_persist
  ok "Hysteria2 端口跳跃已应用（$rng → $port${iface:+, iface=$iface}）"
}

hy2hop_revert() {
  local iface rng changed=0
  iface="$(hy2_default_iface)"
  rng="${HY2_PORTHOP:-}"
  for IPT in iptables ip6tables; do
    command -v "$IPT" >/dev/null 2>&1 || continue
    if [ -n "$rng" ]; then
      "$IPT" -t nat -D PREROUTING -p udp --dport "$rng" -j RD_HY2 2>/dev/null && changed=1 || true
      if [ -n "$iface" ]; then
        "$IPT" -t nat -D PREROUTING -i "$iface" -p udp --dport "$rng" -j RD_HY2 2>/dev/null && changed=1 || true
      fi
    fi
    "$IPT" -t nat -F RD_HY2 2>/dev/null && changed=1 || true
    "$IPT" -t nat -X RD_HY2 2>/dev/null && changed=1 || true
  done
  if [ "$changed" = 1 ]; then
    hy2hop_persist
    ok "Hysteria2 端口跳跃规则已回收"
  else
    log "无端口跳跃规则需回收"
  fi
}

# 持久化：写入 /etc/iptables/rules.v{4,6}，并确保 netfilter-persistent 开机恢复。
hy2hop_persist() {
  mkdir -p /etc/iptables
  if command -v iptables-save >/dev/null 2>&1; then
    iptables-save >/etc/iptables/rules.v4 2>/dev/null || true
  fi
  if command -v ip6tables-save >/dev/null 2>&1; then
    ip6tables-save >/etc/iptables/rules.v6 2>/dev/null || true
  fi
  if ! command -v netfilter-persistent >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get install -y netfilter-persistent iptables-persistent >/dev/null 2>&1 || true
  fi
  if command -v netfilter-persistent >/dev/null 2>&1; then
    systemctl enable netfilter-persistent >/dev/null 2>&1 || true
    netfilter-persistent save >/dev/null 2>&1 || true
  fi
}
