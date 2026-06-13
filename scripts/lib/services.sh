#!/usr/bin/env bash
# 服务与证书相关操作。

svc_enable_restart() {
  local svc="$1"
  systemctl enable "$svc" >/dev/null 2>&1 || true
  systemctl restart "$svc" || die "$svc 重启失败（journalctl -u $svc -n 50）"
}

# 等待 Angie 起来并完成 ACME 签发（轮询 80/8443）。
wait_angie_ready() {
  local i
  log "等待 Angie 启动并签发 ACME 证书（最多 ~60s）…"
  for i in $(seq 1 30); do
    if curl -fsS -o /dev/null --max-time 3 "http://127.0.0.1/" 2>/dev/null \
      || curl -fsSk -o /dev/null --max-time 3 "https://127.0.0.1:8443/" 2>/dev/null; then
      ok "Angie 已就绪"
      return 0
    fi
    sleep 2
  done
  warn "Angie 就绪检测超时——ACME 可能仍在签发，或 80/443 未就绪。请稍后用 'curl -vk https://<domain>/' 复查。"
}

# 生成 Hysteria2 自签证书（EC prime256v1，10 年）。
gen_hy2_cert() {
  local domain="$1" dir=/usr/local/etc/xray
  mkdir -p "$dir"
  if [ -f "$dir/hy2.crt" ] && [ -f "$dir/hy2.key" ]; then
    ok "Hysteria2 证书已存在"; return 0
  fi
  log "生成 Hysteria2 自签证书（CN=$domain）…"
  openssl ecparam -genkey -name prime256v1 -out "$dir/hy2.key" 2>/dev/null
  openssl req -new -x509 -days 3650 -key "$dir/hy2.key" -out "$dir/hy2.crt" \
    -subj "/CN=$domain" 2>/dev/null || die "openssl 生成 hy2 证书失败"
  ok "Hysteria2 证书已生成"
}

# VLESS-TLS 证书置备（best-effort）。
# Angie 内置 ACME 主要服务自身 TLS；要让 Xray 复用，需把 fullchain+key 导出为 PEM 文件。
# 此处尽力而为：若 Angie 已导出则复用；否则给出明确的手动放置指引（不静默失败）。
provision_tls_cert() {
  local domain="$1"
  local cert="/etc/angie/acme/$domain.fullchain.pem"
  local key="/etc/angie/acme/$domain.key"
  mkdir -p /etc/angie/acme
  if [ -f "$cert" ] && [ -f "$key" ]; then
    ok "VLESS-TLS 证书已就绪（$cert）"; return 0
  fi
  local store=""
  for d in /var/lib/angie/acme /var/db/angie/acme /etc/angie/acme-state; do
    [ -d "$d" ] && store="$d" && break
  done
  if [ -n "$store" ]; then
    warn "尝试从 Angie ACME 存储 ($store) 导出证书…"
    cp -a "$store"/*"$domain"* /etc/angie/acme/ 2>/dev/null || true
  fi
  if [ ! -f "$cert" ] || [ ! -f "$key" ]; then
    warn "VLESS-TLS 需要证书文件：$cert 与 $key"
    warn "Angie 内置 ACME 主要面向自身 TLS；供 Xray 使用时需导出 fullchain+key。"
    warn "请用 ACME deploy-hook 导出，或手动放置后执行：systemctl restart xray"
    return 1
  fi
  ok "VLESS-TLS 证书已就绪"
}
