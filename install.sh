#!/usr/bin/env bash
# Reality Deployer —— 一键安装（curl 方式）。
# 优先下载 GitHub Release 预编译 tarball；若没有 Release，自动从 main 源码构建。
# Release 路径不需要 Go；源码 fallback 会临时下载 Go >=1.20（不落盘到系统目录）。
#
#   curl -fsSL https://raw.githubusercontent.com/Boooob626/reality-deployer/main/install.sh | sudo bash
#
# 私有仓库下载：export GH_TOKEN=<PAT> 后再执行，或装 gh 并 gh auth login。
set -euo pipefail

REPO="Boooob626/reality-deployer"
DEST="${DEST:-/opt/reality-deployer}"
REF="${RD_REF:-main}"
GO_BOOTSTRAP_VERSION="${GO_BOOTSTRAP_VERSION:-}"

_rd_die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }
_rd_log() { printf '\033[34m»\033[0m %s\n' "$*" >&2; }
_rd_ok()  { printf '\033[32m✓\033[0m %s\n' "$*" >&2; }

# —— VPS-only 守卫 ——
[ "$(uname -s)" = "Linux" ] || _rd_die "仅支持 Linux VPS（当前: $(uname -s)）。"
[ "$(id -u)" -eq 0 ] || _rd_die "请用 root 运行：curl ... | sudo bash"
case "$DEST" in
  ""|"/"|"/opt"|"/usr"|"/usr/local"|"/etc"|"/var")
    _rd_die "DEST 不安全：$DEST"
    ;;
esac

install_pkgs() {
  [ "$#" -eq 0 ] && return 0
  command -v apt-get >/dev/null 2>&1 || _rd_die "仅支持 Debian/Ubuntu VPS（需要 apt-get）。"
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y "$@"
}

missing=()
command -v curl >/dev/null 2>&1 || missing+=(curl)
command -v tar  >/dev/null 2>&1 || missing+=(tar)
[ -f /etc/ssl/certs/ca-certificates.crt ] || missing+=(ca-certificates)
install_pkgs "${missing[@]}"

# 架构 → asset 名
case "$(uname -m)" in
  x86_64)        ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) _rd_die "不支持的架构: $(uname -m)（仅 amd64/arm64）" ;;
esac
ASSET="reality-deployer-linux-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
curl_args=(-fsSL)
[ -n "${GH_TOKEN:-}" ] && curl_args+=(-H "Authorization: token $GH_TOKEN")

download() {
  local url="$1" out="$2"
  curl "${curl_args[@]}" -o "$out" "$url"
}

install_payload() {
  _rd_log "解压到 $DEST …"
  rm -rf "$DEST"; mkdir -p "$DEST"
  tar -xzf "$1" -C "$DEST"
  chmod +x "$DEST/reality-deployer" "$DEST/deploy.sh" 2>/dev/null || true
}

go_version_ok() {
  command -v go >/dev/null 2>&1 || return 1
  local parsed major minor
  parsed="$(go version 2>/dev/null | sed -nE 's/.* go([0-9]+)\.([0-9]+).*/\1 \2/p')"
  [ -n "$parsed" ] || return 1
  # shellcheck disable=SC2086
  set -- $parsed
  major="$1"; minor="$2"
  [ "$major" -gt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -ge 20 ]; }
}

ensure_go() {
  if go_version_ok; then
    _rd_log "使用 $(go version)"
    return 0
  fi

  local gov="$GO_BOOTSTRAP_VERSION"
  if [ -z "$gov" ]; then
    gov="$(curl "${curl_args[@]}" "https://go.dev/VERSION?m=text" 2>/dev/null \
      | sed -nE 's/^go([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/p' \
      | head -n1 || true)"
  fi
  [ -n "$gov" ] || gov="1.22.12"

  local go_url="https://go.dev/dl/go${gov}.linux-${ARCH}.tar.gz"
  _rd_log "未发现 Go >=1.20，下载临时 Go ${gov} 用于构建 …"
  download "$go_url" "$TMP/go.tar.gz" || _rd_die "下载 Go 失败：$go_url"
  tar -xzf "$TMP/go.tar.gz" -C "$TMP"
  export PATH="$TMP/go/bin:$PATH"
  go_version_ok || _rd_die "临时 Go 不可用：$(go version 2>/dev/null || true)"
}

install_from_source() {
  local src_url="${RD_SOURCE_URL:-https://github.com/${REPO}/archive/refs/heads/${REF}.tar.gz}"
  _rd_log "Release asset 不可用，改从源码构建：$src_url"
  download "$src_url" "$TMP/source.tar.gz" || _rd_die "下载源码失败：$src_url"
  mkdir -p "$TMP/src"
  tar -xzf "$TMP/source.tar.gz" -C "$TMP/src" --strip-components=1

  ensure_go
  _rd_log "构建 reality-deployer (${ARCH}) …"
  (cd "$TMP/src" && CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags="-s -w" -o "$TMP/reality-deployer" ./cmd/deploy) \
    || _rd_die "源码构建失败"

  _rd_log "安装到 $DEST …"
  rm -rf "$DEST"; mkdir -p "$DEST"
  install -m 0755 "$TMP/reality-deployer" "$DEST/reality-deployer"
  install -m 0755 "$TMP/src/deploy.sh" "$TMP/src/install.sh" "$DEST/"
  cp -R "$TMP/src/scripts" "$DEST/scripts"
  chmod +x "$DEST/scripts/install.sh" "$DEST/scripts/uninstall.sh" 2>/dev/null || true
}

installed=0
if [ "${RD_INSTALL_FROM_SOURCE:-0}" != "1" ]; then
  _rd_log "下载 $URL …"
  if download "$URL" "$TMP/$ASSET"; then
    install_payload "$TMP/$ASSET"
    installed=1
  elif command -v gh >/dev/null 2>&1; then
    _rd_log "Release curl 失败，改用 gh release download …"
    if gh release download -R "$REPO" -p "$ASSET" -D "$TMP"; then
      install_payload "$TMP/$ASSET"
      installed=1
    fi
  fi
fi

if [ "$installed" -ne 1 ]; then
  if command -v gh >/dev/null 2>&1; then
    _rd_log "没有可用 Release asset，继续使用源码 fallback。"
  fi
  install_from_source
fi

chmod +x "$DEST/reality-deployer" "$DEST/deploy.sh" 2>/dev/null || true

cd "$DEST"
_rd_ok "安装完成，启动向导…"
# curl|bash 时本脚本的 stdin 是管道（已耗尽→EOF）；把向导的 stdin 重新接到控制终端，
# 否则向导读取交互输入会立即 EOF 死循环。
if [ -t 0 ]; then
  exec ./deploy.sh "$@"
elif [ -r /dev/tty ]; then
  exec ./deploy.sh "$@" </dev/tty
else
  _rd_die "当前没有可交互终端（/dev/tty 不可用）。请登录 VPS 后运行：sudo $DEST/deploy.sh"
fi
