#!/usr/bin/env bash
# Reality Deployer —— 一键安装（curl 方式）。
# 从 GitHub Release 下载预编译 tarball（含二进制 + scripts/ + deploy.sh），解压即用。
# VPS 上不编译、不需要 make / Go 工具链。
#
#   curl -fsSL https://raw.githubusercontent.com/Boooob626/reality-deployer/main/install.sh | sudo bash
#
# 私有仓库下载：export GH_TOKEN=<PAT> 后再执行，或装 gh 并 gh auth login。
set -euo pipefail

REPO="Boooob626/reality-deployer"
DEST="${DEST:-/opt/reality-deployer}"

_rd_die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }
_rd_log() { printf '\033[34m»\033[0m %s\n' "$*" >&2; }
_rd_ok()  { printf '\033[32m✓\033[0m %s\n' "$*" >&2; }

# —— VPS-only 守卫 ——
[ "$(uname -s)" = "Linux" ] || _rd_die "仅支持 Linux VPS（当前: $(uname -s)）。"
[ "$(id -u)" -eq 0 ] || _rd_die "请用 root 运行：curl ... | sudo bash"

command -v curl >/dev/null 2>&1 || { apt-get update -qq && apt-get install -y curl; }
command -v tar  >/dev/null 2>&1 || apt-get install -y tar

# 架构 → asset 名
case "$(uname -m)" in
  x86_64)        ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) _rd_die "不支持的架构: $(uname -m)（仅 amd64/arm64）" ;;
esac
ASSET="reality-deployer-linux-${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
_rd_log "下载 $URL …"

# 下载：curl（公开仓库直连；私有仓库可设 GH_TOKEN）；失败回退 gh
curl_args=(-fsSL)
[ -n "${GH_TOKEN:-}" ] && curl_args+=(-H "Authorization: token $GH_TOKEN")
if ! curl "${curl_args[@]}" -o "$TMP/$ASSET" "$URL"; then
  if command -v gh >/dev/null 2>&1; then
    _rd_log "curl 失败，改用 gh release download …"
    gh release download -R "$REPO" -p "$ASSET" -D "$TMP" || _rd_die "下载失败（检查网络 / 仓库可访问性）"
  else
    _rd_die "下载失败（公开仓库应可直接 curl；私有仓库请 export GH_TOKEN=<PAT> 或安装 gh 并 gh auth login）"
  fi
fi

_rd_log "解压到 $DEST …"
rm -rf "$DEST"; mkdir -p "$DEST"
tar -xzf "$TMP/$ASSET" -C "$DEST"
chmod +x "$DEST/reality-deployer" "$DEST/deploy.sh" 2>/dev/null || true

cd "$DEST"
_rd_ok "安装完成，启动向导…"
# curl|bash 时本脚本的 stdin 是管道（已耗尽→EOF）；把向导的 stdin 重新接到控制终端，
# 否则向导读取交互输入会立即 EOF 死循环。
if [ -t 0 ]; then
  exec ./deploy.sh "$@"
else
  exec ./deploy.sh "$@" </dev/tty
fi
