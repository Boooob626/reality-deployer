#!/usr/bin/env bash
# 包安装：Xray（官方脚本）+ Angie（官方仓库）。
# Xray 官方脚本会附带 geoip.dat / geosite.dat，正是 routing 规则所需。

apt_refresh() {
  apt-get update -qq
}

cleanup_legacy_angie_repo() {
  for f in /etc/apt/sources.list.d/angie.list /etc/apt/sources.list.d/angie.sources; do
    [ -f "$f" ] || continue
    if grep -Eq 'download\.angie\.software/angie/(debian|ubuntu)([[:space:]]|$)' "$f"; then
      warn "移除旧 Angie apt 源：$f"
      rm -f "$f"
    fi
  done
}

angie_repo_base() {
  [ -n "${DISTRO_ID:-}" ] || die "发行版 ID 未检测，请先调用 detect_distro"
  [ -n "${DISTRO_VERSION_ID:-}" ] || die "发行版版本未检测，无法配置 Angie 仓库"
  printf 'https://download.angie.software/angie/%s/%s' "$DISTRO_ID" "$DISTRO_VERSION_ID"
}

angie_repo_release_url() {
  local base="$1" suite="$2"
  printf '%s/dists/%s/Release' "$base" "$suite"
}

# 安装 Xray-core（幂等）。
install_xray() {
  if command -v xray >/dev/null 2>&1 && xray version >/dev/null 2>&1; then
    ok "Xray 已安装：$(xray version | head -1)"
    return 0
  fi
  log "安装 Xray-core（官方脚本）…"
  local installer
  installer="$(mktemp)"
  if ! curl -fsSL https://github.com/XTLS/Xray-install/raw/main/install-release.sh -o "$installer"; then
    rm -f "$installer"
    die "下载 Xray 安装脚本失败"
  fi
  if ! bash "$installer" install; then
    rm -f "$installer"
    die "Xray 安装失败"
  fi
  rm -f "$installer"
  command -v xray >/dev/null 2>&1 || die "Xray 安装后仍找不到 xray 命令"
  ok "Xray 安装完成"
}

# 安装 Angie（官方仓库，幂等）。
install_angie() {
  if command -v angie >/dev/null 2>&1; then
    ok "Angie 已安装：$(angie -v 2>&1 | head -1)"
    return 0
  fi
  log "安装 Angie（官方仓库）…"
  cleanup_legacy_angie_repo
  pkg_ensure ca-certificates curl gnupg
  local keyring=/usr/share/keyrings/angie-signing.gpg
  mkdir -p "$(dirname "$keyring")"
  curl -fsSL https://angie.software/keys/angie-signing.gpg \
    | gpg --dearmor --yes -o "$keyring" || die "下载 Angie 签名密钥失败"
  local repo_base release_url
  repo_base="$(angie_repo_base)"
  release_url="$(angie_repo_release_url "$repo_base" "$DISTRO_CODENAME")"
  if ! curl -fsI "$release_url" >/dev/null 2>&1; then
    die "Angie 仓库不支持当前发行版：$DISTRO_ID $DISTRO_VERSION_ID ($DISTRO_CODENAME)。缺少 $release_url"
  fi
  rm -f /etc/apt/sources.list.d/angie.list /etc/apt/sources.list.d/angie.sources
  echo "deb [signed-by=$keyring] $repo_base $DISTRO_CODENAME main" \
    > /etc/apt/sources.list.d/angie.list
  apt_refresh
  apt-get install -y angie || die "Angie 安装失败"
  # ACME 模块部分发行版独立打包；缺失时核心可能已内置，由 angie -t 兜底校验。
  apt-get install -y angie-module-acme 2>/dev/null || true
  ok "Angie 安装完成"
}

# 探测 Angie 站点 include 目录（http.d 或 conf.d）。
angie_conf_dir() {
  for d in /etc/angie/http.d /etc/angie/conf.d; do
    if [ -d "$d" ]; then echo "$d"; return 0; fi
  done
  # 兜底：检查主配置 include 指令
  if grep -rqE 'include\s+.*conf\.d/\*\.conf' /etc/angie/angie.conf 2>/dev/null; then
    echo /etc/angie/conf.d; return 0
  fi
  echo /etc/angie/http.d
}
