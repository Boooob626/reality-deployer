#!/usr/bin/env bash
# Reality Deployer —— 卸载（按 apply.env 精确回收，默认保留已装软件包与证书）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
for lib in "$SCRIPT_DIR"/lib/*.sh; do
  # shellcheck source=/dev/null
  source "$lib"
done

require_root

MANIFEST_DIR="${MANIFEST_DIR:-/var/lib/reality-deployer}"
APPLY_ENV="$MANIFEST_DIR/apply.env"
if [ ! -f "$APPLY_ENV" ]; then
  die "找不到 $APPLY_ENV，无法精确回收。可手动 systemctl disable --now xray angie。"
fi
# shellcheck source=/dev/null
source "$APPLY_ENV"
STAGING="${STAGING:-$MANIFEST_DIR/staging}"
: "${DOMAIN:?apply.env 缺少 DOMAIN}"
ANGIE_DIR="$(angie_conf_dir 2>/dev/null || echo /etc/angie/http.d)"

log "卸载 Reality Deployer 部署（domain=$DOMAIN）"

# 1) 停服务
systemctl disable --now xray 2>/dev/null || true
systemctl disable --now angie 2>/dev/null || true

# 2) 回收 ufw 规则
ufw_revert_rules

# 2b) 回收 Hysteria2 端口跳跃 iptables 规则
hy2hop_revert

# 3) 删除部署产生的配置/文件
remove_file /usr/local/etc/xray/config.json
remove_file "$ANGIE_DIR/$DOMAIN.conf"
remove_file /etc/systemd/system/xray.service
remove_file /usr/local/etc/xray/hy2.crt
remove_file /usr/local/etc/xray/hy2.key
if [ "${KERNEL_LOW_LATENCY:-0}" = "1" ]; then
  remove_file /etc/sysctl.d/99-reality-deployer.conf
  sysctl --system >/dev/null 2>&1 || true
fi
[ -d "/var/www/$DOMAIN" ] && rm -rf "/var/www/$DOMAIN" && log "已移除 /var/www/$DOMAIN"

systemctl daemon-reload

# 4) 是否删除 manifest/staging 与软件包
if [ "${PURGE_ALL:-0}" = "1" ]; then
  rm -rf "$MANIFEST_DIR" && log "已移除 $MANIFEST_DIR"
  apt-get purge -y xray angie angie-module-acme 2>/dev/null || true
  apt-get autoremove -y 2>/dev/null || true
  ok "已彻底卸载（含软件包与证书目录）"
else
  ok "已卸载部署。保留：软件包(xray/angie)、ACME 证书、$MANIFEST_DIR 记录。"
  log "彻底清理（含 apt purge + 删除记录）：PURGE_ALL=1 sudo bash $0"
fi
