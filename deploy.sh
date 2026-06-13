#!/usr/bin/env bash
# Reality Deployer —— VPS 入口。
# 二进制随 Release tarball 下发，**不在 VPS 上编译**（无需 make / Go 工具链）。
#
# 用法：
#   sudo ./deploy.sh                  # 向导 → 询问是否应用（默认）
#   sudo ./deploy.sh wizard|apply|uninstall
#   ./deploy.sh  status|export
#
# 一键安装（见 install.sh）：
#   curl -fsSL https://raw.githubusercontent.com/Boooob626/reality-deployer/main/install.sh | sudo bash
set -euo pipefail

# —— VPS-only 守卫 ——
[ "$(uname -s)" = "Linux" ] || { printf '\033[31m✗\033[0m 仅支持 Linux VPS（当前: %s）\n' "$(uname -s)" >&2; exit 1; }

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export REALITY_DEPLOYER_ROOT="$HERE"

_rd_die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# 定位二进制（tarball 解压后位于 $HERE/reality-deployer）
BIN=""
for c in \
  "$HERE/reality-deployer" \
  "$HERE/reality-deployer-linux-amd64" \
  "$HERE/reality-deployer-linux-arm64" \
  "$(command -v reality-deployer 2>/dev/null || true)"; do
  if [ -n "$c" ] && [ -x "$c" ]; then BIN="$c"; break; fi
done

[ -n "$BIN" ] || _rd_die "找不到 reality-deployer 二进制。
一键安装：
  curl -fsSL https://raw.githubusercontent.com/Boooob626/reality-deployer/main/install.sh | sudo bash"

SUBCMD="${1:-wizard}"
case "$SUBCMD" in
  apply|uninstall)
    [ "$(id -u)" -eq 0 ] || _rd_die "$SUBCMD 需要 root，请用 sudo。"
    exec "$BIN" "$SUBCMD" "${@:2}"
    ;;
  status|export)
    exec "$BIN" "$SUBCMD" "${@:2}"
    ;;
  wizard|help|-h|--help)
    ;;
  *)
    echo "用法: $0 [wizard|apply|uninstall|status|export]" >&2
    exit 2
    ;;
esac

# 默认：向导 → 询问是否应用
"$BIN" wizard
echo
if [ "$(id -u)" -eq 0 ]; then
  read -r -p "立即应用到系统（安装/配置/启动/签发证书）？[y/N] " ans
  case "$ans" in
    y|Y|yes) exec "$BIN" apply ;;
    *) echo "稍后应用：sudo $0 apply" ;;
  esac
else
  echo "向导已完成配置渲染。应用：sudo $0 apply"
fi
