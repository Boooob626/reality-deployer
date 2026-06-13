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

angie_acme_client_id() {
  local id="${1,,}"
  id="${id//./_}"
  id="${id//-/_}"
  printf 'le_%s\n' "$id"
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

# VLESS-TLS 证书置备。
# Angie ACME 证书默认位于 acme_client_path/<client>/{certificate.pem,private.key}。
# Xray 需要稳定文件路径，因此复制一份到 /etc/angie/acme/<domain>.*。
provision_tls_cert() {
  local domain="$1" client cert key wait_until src_dir
  client="$(angie_acme_client_id "$domain")"
  cert="/etc/angie/acme/$domain.fullchain.pem"
  key="/etc/angie/acme/$domain.key"
  mkdir -p /etc/angie/acme
  if [ -f "$cert" ] && [ -f "$key" ]; then
    ok "VLESS-TLS 证书已就绪（$cert）"; return 0
  fi

  wait_until=$((SECONDS + ${TLS_WAIT_SECONDS:-120}))
  while :; do
    for src_dir in \
      "/etc/angie/acme/$client" \
      "/var/lib/angie/acme/$client" \
      "/var/db/angie/acme/$client" \
      "/etc/angie/acme-state/$client"; do
      if [ -f "$src_dir/certificate.pem" ] && [ -f "$src_dir/private.key" ]; then
        install -m 0644 "$src_dir/certificate.pem" "$cert"
        install -m 0600 "$src_dir/private.key" "$key"
        ok "VLESS-TLS 证书已导出（$cert）"
        return 0
      fi
    done
    [ "$SECONDS" -ge "$wait_until" ] && break
    sleep 2
  done

  warn "VLESS-TLS 需要证书文件：$cert 与 $key"
  warn "未在 Angie ACME 存储中找到 $client/certificate.pem 与 private.key。"
  return 1
}
