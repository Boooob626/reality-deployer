#!/usr/bin/env bash
# 通用工具：彩色日志、root 检查、幂等的文件/包操作。
# 被 install.sh / uninstall.sh source；本文件不含顶层副作用逻辑。
set -eo pipefail

if [ -t 1 ]; then
  RED=$'\033[31m'; GREEN=$'\033[32m'; YEL=$'\033[33m'; BLUE=$'\033[34m'; NC=$'\033[0m'
else
  RED=""; GREEN=""; YEL=""; BLUE=""; NC=""
fi

log()  { printf "%s»%s %s\n" "$BLUE" "$NC" "$*"; }
ok()   { printf "%s✓%s %s\n" "$GREEN" "$NC" "$*"; }
warn() { printf "%s⚠%s %s\n" "$YEL" "$NC" "$*" >&2; }
die()  { printf "%s✗%s %s\n" "$RED" "$NC" "$*" >&2; exit 1; }

require_root() {
  [ "$(id -u)" -eq 0 ] || die "请以 root 运行（sudo）。"
}

# 幂等安装 deb 包。
pkg_ensure() {
  export DEBIAN_FRONTEND=noninteractive
  local missing=()
  for p in "$@"; do
    dpkg -s "$p" >/dev/null 2>&1 || missing+=("$p")
  done
  [ "${#missing[@]}" -eq 0 ] && return 0
  apt-get install -y "${missing[@]}"
}

# 备份已存在的文件（带时间戳）。
backup_file() {
  [ -f "$1" ] || return 0
  local ts; ts="$(date +%Y%m%d-%H%M%S)"
  cp -a "$1" "$1.bak.$ts"
  log "已备份 $1 → $1.bak.$ts"
}

# install_file SRC DEST [MODE]：创建父目录并写入。
install_file() {
  local src="$1" dest="$2" mode="${3:-0644}"
  [ -f "$src" ] || die "install_file: 源不存在: $src"
  mkdir -p "$(dirname "$dest")"
  install -m "$mode" "$src" "$dest"
  log "已写入 $dest"
}

# 移除文件（幂等）。
remove_file() {
  [ -e "$1" ] && rm -f "$1" && log "已移除 $1" || true
}
