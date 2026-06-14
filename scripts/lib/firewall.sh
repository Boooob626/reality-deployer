#!/usr/bin/env bash
# ufw 规则管理。规则实体由 Go 渲染到 $STAGING/ufw_rules.sh（ufw_apply / ufw_revert）。

ensure_ufw() {
  command -v ufw >/dev/null 2>&1 || apt-get install -y ufw
}

sanitize_ufw_rules() {
  local rules="$1"
  # Older generated rules used invalid "ufw --force allow/delete" syntax.
  sed -i \
    -e 's/ufw --force allow /ufw allow /g' \
    -e 's/ufw --force delete allow /ufw delete allow /g' \
    "$rules"
}

ufw_apply_rules() {
  local rules="${STAGING}/ufw_rules.sh"
  [ -f "$rules" ] || die "找不到 $rules（请先运行 wizard）"
  ensure_ufw
  sanitize_ufw_rules "$rules"
  ufw --force default deny incoming >/dev/null
  ufw --force default allow outgoing >/dev/null
  # shellcheck source=/dev/null
  source "$rules"
  ufw_apply
  ufw --force enable >/dev/null
  ok "ufw 规则已应用（含 SSH/80/443/可选 hy2）"
}

ufw_revert_rules() {
  local rules="${STAGING}/ufw_rules.sh"
  if [ ! -f "$rules" ]; then warn "无 $rules，跳过 ufw 回收"; return 0; fi
  ensure_ufw
  sanitize_ufw_rules "$rules"
  # shellcheck source=/dev/null
  source "$rules"
  ufw_revert || true
  ok "ufw 规则已回收"
}
