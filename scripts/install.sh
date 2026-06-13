#!/usr/bin/env bash
# Reality Deployer —— 应用部署到系统（幂等）。
# 流程：装包 → 落位配置 → (hy2 自签证书 / tls 证书) → ufw → systemd+angie+xray → 校验。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for lib in "$SCRIPT_DIR"/lib/*.sh; do
  # shellcheck source=/dev/null
  source "$lib"
done

require_root
detect_distro

MANIFEST_DIR="${MANIFEST_DIR:-/var/lib/reality-deployer}"
APPLY_ENV="$MANIFEST_DIR/apply.env"
[ -f "$APPLY_ENV" ] || die "找不到 $APPLY_ENV（请先运行 'reality-deployer wizard' 或 'sudo ./deploy.sh'）"
# shellcheck source=/dev/null
source "$APPLY_ENV"
STAGING="${STAGING:-$MANIFEST_DIR/staging}"
[ -d "$STAGING" ] || die "staging 目录不存在：$STAGING（请先运行 wizard）"

: "${DOMAIN:?apply.env 缺少 DOMAIN}"
: "${SSH_PORT:?apply.env 缺少 SSH_PORT}"

log "域名=$DOMAIN  公网IP=${PUBLIC_IP:-未知}"
log "组合: vless-reality=${ENABLE_VLESS_REALITY:-0} vless-tls=${ENABLE_VLESS_TLS:-0} hysteria2=${ENABLE_HYSTERIA2:-0}"

# 1) 基础包 + Xray + Angie
pkg_ensure curl ca-certificates openssl ufw
install_xray
install_angie

# 2) 落位配置文件
ANGIE_DIR="$(angie_conf_dir)"

backup_file /usr/local/etc/xray/config.json || true
install_file "$STAGING/xray/config.json"          /usr/local/etc/xray/config.json 0644
install_file "$STAGING/angie/$DOMAIN.conf"        "$ANGIE_DIR/$DOMAIN.conf"       0644
install_file "$STAGING/decoy/index.html"          "/var/www/$DOMAIN/index.html"   0644
install_file "$STAGING/systemd/xray.service"      /etc/systemd/system/xray.service 0644
mkdir -p /etc/angie/acme

# 3) Hysteria2 自签证书
if [ "${ENABLE_HYSTERIA2:-0}" = "1" ]; then
  gen_hy2_cert "$DOMAIN"
fi

# 4) VLESS-TLS 证书（best-effort；REALITY 组合无需证书文件）
if [ "${ENABLE_VLESS_TLS:-0}" = "1" ]; then
  provision_tls_cert "$DOMAIN" || warn "VLESS-TLS 证书暂缺，xray 可能启动失败——按提示放置证书后 'systemctl restart xray'"
fi

# 5) geo 数据（routing 的 geosite/geoip 规则依赖）
if [ ! -f /usr/local/share/xray/geosite.dat ]; then
  warn "缺少 /usr/local/share/xray/geosite.dat——CN 路由规则将不生效。Xray 官方脚本通常会附带，若缺失请手动补齐。"
fi

# 6) ufw（先放行 SSH/80/443/hy2，再 enable）
ufw_apply_rules

# 6b) Hysteria2 端口跳跃（iptables REDIRECT 区间 → 监听端口）
hy2hop_apply

# 7) systemd + Angie（ACME 随启动签发）→ 再起 Xray
systemctl daemon-reload
svc_enable_restart angie
wait_angie_ready
svc_enable_restart xray

# 8) 校验
log "校验配置语法…"
angie -t || die "angie 配置校验失败（angie -t）"
/usr/local/bin/xray run -test -config /usr/local/etc/xray/config.json \
  || die "xray 配置校验失败"

ok "部署完成"
echo
log "ufw 状态："
ufw status verbose
echo
log "客户端连接链接（reality-deployer export 可重新导出）："
REALITY_DEPLOYER_ROOT="${REALITY_DEPLOYER_ROOT:-$SCRIPT_DIR/..}" \
  "$(command -v reality-deployer || echo "$SCRIPT_DIR/../dist/reality-deployer")" export || true
echo
ok "提示：在客户端导入上方链接；浏览器访问 https://$DOMAIN/ 应看到伪装站点（REALITY own_domain 源时）。"
