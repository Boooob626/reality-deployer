#!/usr/bin/env bash
# 发行版检测：仅 Debian/Ubuntu。
detect_distro() {
  [ -f /etc/os-release ] || die "找不到 /etc/os-release，无法识别发行版"
  # shellcheck disable=SC1091
  . /etc/os-release
  DISTRO_ID="${ID:-}"
  DISTRO_CODENAME="${VERSION_CODENAME:-}"
  DISTRO_VERSION_ID="${VERSION_ID:-}"
  case "$DISTRO_ID" in
    debian|ubuntu) ;;
    *) die "仅支持 Debian/Ubuntu，当前: ${DISTRO_ID:-未知}（${PRETTY_NAME:-}）" ;;
  esac
  log "发行版: ${PRETTY_NAME:-$DISTRO_ID $DISTRO_CODENAME}"
}
